package insights

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Response struct {
	Code int
	Data []byte
}

func (r *Response) String() string {
	return fmt.Sprintf("%d: %s", r.Code, string(r.Data))
}

// Ingress represents the service where we upload archives.
var Ingress = newService(
	&url.URL{Scheme: "https", Host: "cert.console.redhat.com:443"},
	"api/ingress/v1",
)

// service provides an abstraction over APIs
// TODO Support other means of authentication
type service struct {
	URL               *url.URL
	Path              string
	ClientCertificate string
	ClientKey         string
	Proxy             *url.URL
}

func newService(address *url.URL, path string) *service {
	return &service{URL: address, Path: path}
}

func (s *service) SetCertAuth(certificate, key string) error {
	s.ClientCertificate = certificate
	s.ClientKey = key
	return nil
}

func (s *service) SetProxy(proxy *url.URL) error {
	s.Proxy = proxy
	return nil
}

func (s *service) String() string {
	return fmt.Sprintf("%s://%s/%s", s.URL.Scheme, s.URL.Host, s.Path)
}

func (s *service) newClient() (*http.Client, error) {
	// TODO Detect cert auth in some better way
	var clientCertificates []tls.Certificate
	cert, err := tls.LoadX509KeyPair(s.ClientCertificate, s.ClientKey)
	if err == nil {
		clientCertificates = append(clientCertificates, cert)
		slog.Debug("using client certificate", "path", s.ClientCertificate)
	} else {
		slog.Debug("not using client certificate", "err", err)
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		slog.Error("cannot load system certificates", slog.String("error", err.Error()))
		return nil, err
	}

	caCert, err := os.ReadFile(s.ClientCertificate)
	if err == nil {
		pool.AppendCertsFromPEM(caCert)
	}

	tlsConfig := &tls.Config{RootCAs: pool, Certificates: clientCertificates}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	if s.Proxy != nil {
		slog.Debug("using proxy", slog.String("url", s.Proxy.String()))
		transport.Proxy = http.ProxyURL(s.Proxy)
	}

	return &http.Client{Transport: transport}, nil
}

func (s *service) Call(
	method string,
	endpoint string,
	parameters url.Values,
	headers map[string][]string,
	body *bytes.Buffer,
) (*Response, error) {
	fullUrl := fmt.Sprintf("%s/%s?%s", s.String(), endpoint, parameters.Encode())

	if body == nil {
		body = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, fullUrl, body)
	if err != nil {
		slog.Error("cannot construct request", slog.String("err", err.Error()))
		return nil, errors.New("cannot construct request")
	}

	for key, value := range headers {
		req.Header[key] = value
	}
	if _, ok := req.Header["Accept"]; !ok {
		// Default to requesting JSON
		req.Header.Set("Accept", "application/json")
	}

	client, err := s.newClient()
	if err != nil {
		return nil, err
	}

	{
		// Log request structure
		valuesToLog := []any{
			slog.String("method", method),
			slog.String("URL", fullUrl),
			slog.Any("headers", req.Header),
		}
		if s.Proxy != nil {
			valuesToLog = append(valuesToLog, slog.String("proxy", s.Proxy.String()))
		}
		slog.Debug("built request", valuesToLog...)
	}

	now := time.Now()
	resp, err := client.Do(req)
	delta := time.Since(now)

	if err != nil {
		slog.Error("could not make request", slog.String("error", err.Error()))
		return nil, errors.New("could not make API request")
	}
	slog.Debug("response received", slog.Int("code", resp.StatusCode), slog.Duration("rtt", delta))
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("could not read response body", slog.String("error", err.Error()))
		return nil, errors.New("could not read response body")
	}

	return &Response{Code: resp.StatusCode, Data: response}, nil
}

func Upload(archive, contentType string) error {
	slog.Debug("uploading", slog.String("archive", archive), slog.String("content-type", contentType))

	formData := new(bytes.Buffer)
	form := multipart.NewWriter(formData)

	archiveHeader := make(textproto.MIMEHeader)
	archiveHeader.Set(
		"Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", filepath.Base(archive)),
	)
	archiveHeader.Set("Content-Type", contentType)

	archiveField, err := form.CreatePart(archiveHeader)
	if err != nil {
		slog.Error("could not add archive to request", slog.Any("error", err))
		return errors.New("failed to include archive in upload")
	}

	archiveDescriptor, err := os.Open(archive)
	if err != nil {
		slog.Error("could not open archive", slog.Any("error", err))
		return errors.New("failed to include archive in upload")
	}
	defer archiveDescriptor.Close()

	_, err = io.Copy(archiveField, archiveDescriptor)
	if err != nil {
		slog.Error("could not read archive", slog.Any("error", err))
		return errors.New("failed to include archive in upload")
	}
	form.Close()

	headers := make(map[string][]string)
	headers["Content-Type"] = []string{form.FormDataContentType()}

	response, err := Ingress.Call("POST", "upload", url.Values{}, headers, formData)
	if err != nil {
		slog.Error("could not upload archive", slog.String("error", err.Error()))
		return errors.New("failed to upload archive")
	}

	if response.Code/100 != 2 {
		slog.Error("server rejected the archive", slog.Int("code", response.Code), slog.Any("response", stringifyData(response.Data)))
		return errors.New("failed to upload archive")
	}

	return nil
}

// stringifyData takes in a byte slice and converts it to string.
//
// Since the data may contain anything, bytes outside printable ASCII are
// converted to simplified representation.
func stringifyData(data []byte) string {
	result := make([]byte, len(data))

	unprintable := 0
	for _, char := range data {
		isPrintable := false
		if char == '\n' || char == '\r' || (char >= ' ' && char < 127) {
			isPrintable = true
		}

		if isPrintable && unprintable > 0 {
			result = append(result, '.')
		}
		if isPrintable {
			result = append(result, char)
			unprintable = 0
		} else {
			unprintable++
		}
	}

	return strings.TrimSpace(string(result))
}
