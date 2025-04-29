package insights

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

var CONFIGURATIONS_DIR string = "."
var COLLECTIONS_DIR string = "/tmp/"
var COLLECTIONS_DIR_PERMISSIONS os.FileMode = 0750
var COLLECTIONS_DIR_ENVVAR = "COLLECTION_DIRECTORY"

type Collector struct {
	Meta struct {
		ID   string `toml:"id" json:"id"`
		Name string `toml:"name" json:"name"`
	} `toml:"meta" json:"meta"`
	Exec struct {
		Shell       string `toml:"shell" json:"shell"`
		ContentType string `toml:"content_type" json:"content_type"`
	} `toml:"exec" json:"exec"`
}

// newCollectorFromPath loads a collector definition from path.
func newCollectorFromPath(path string) (*Collector, error) {
	path, _ = filepath.Abs(path)
	_, err := os.Stat(path)
	if err != nil {
		slog.Error("no such collector", "path", path)
		return nil, errors.New("no such collector")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("cannot read collector configuration", "path", path)
		return nil, fmt.Errorf("cannot read collector configuration from '%s'", path)
	}
	return newCollectorFromConfiguration(string(data))
}

// newCollectorFromConfiguration parses the content of the configuration file into Collector.
func newCollectorFromConfiguration(config string) (*Collector, error) {
	var cc Collector
	_, err := toml.Decode(config, &cc)
	if err != nil {
		slog.Error("cannot parse collector configuration")
		return nil, fmt.Errorf("cannot parse collector configuration")
	}
	slog.Debug("collector parsed", "id", cc.Meta.ID)
	return &cc, nil
}

// ensureCollectorsDirectory raises an error if CONFIGURATIONS_DIR does not exist.
func ensureCollectorsDirectory() error {
	if _, err := os.Stat(CONFIGURATIONS_DIR); os.IsNotExist(err) {
		log.Printf("configuration directory '%s' not found", CONFIGURATIONS_DIR)
		return fmt.Errorf("configuration directory '%s' not found", CONFIGURATIONS_DIR)
	}
	return nil
}

// GetCollector loads collector definition from CONFIGURATIONS_DIR.
func GetCollector(id string) (*Collector, error) {
	if err := ensureCollectorsDirectory(); err != nil {
		return nil, err
	}
	return newCollectorFromPath(filepath.Join(CONFIGURATIONS_DIR, id+".toml"))
}

// GetCollectors loads collector definitions from CONFIGURATIONS_DIR.
func GetCollectors() ([]*Collector, error) {
	if err := ensureCollectorsDirectory(); err != nil {
		return nil, err
	}

	configurations, err := filepath.Glob(filepath.Join(CONFIGURATIONS_DIR, "*.toml"))
	if err != nil {
		log.Printf("cannot scan %s", CONFIGURATIONS_DIR)
		return nil, fmt.Errorf("cannot scan %s", CONFIGURATIONS_DIR)
	}

	var collectors []*Collector
	for _, file := range configurations {
		config, err := newCollectorFromPath(file)
		if err != nil {
			log.Printf("collector '%s' is malformed, skipping: %v", file, err)
			continue
		}
		collectors = append(collectors, config)
	}
	return collectors, nil
}

func generateCollectionDirectory(collector *Collector) (string, error) {
	path := filepath.Join(COLLECTIONS_DIR, collector.Meta.ID+"-"+strconv.FormatInt(time.Now().Unix(), 10))
	if err := os.MkdirAll(path, COLLECTIONS_DIR_PERMISSIONS); err != nil {
		slog.Error("cannot create collector directory", "id", collector.Meta.ID, "err", "err")
		return "", fmt.Errorf("cannot create collector directory")
	}
	slog.Debug("generated collection directory", "path", path)
	return path, nil
}

// Collect instructs the collector to dump data into a temporary directory created inside COLLECTIONS_DIR.
//
// Returns path to the temporary directory, where the data has been dumped, or an error.
func Collect(collector *Collector) (string, error) {
	cmd := exec.Command(
		strings.Split(collector.Exec.Shell, " ")[0],
		strings.Split(collector.Exec.Shell, " ")[1:]...,
	)
	for _, variable := range os.Environ() {
		cmd.Env = append(cmd.Env, variable)
	}
	tempdir, err := generateCollectionDirectory(collector)
	if err != nil {
		return "", err
	}
	cmd.Env = append(cmd.Env, COLLECTIONS_DIR_ENVVAR+"="+tempdir)

	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	slog.Debug("executing", "cmd", cmd)
	err = cmd.Run()
	if err != nil {
		slog.Error("could not run collector", "err", err, "stderr", stderrBuffer.String())
		return "", fmt.Errorf("could not run collector: %v", err)
	}

	return tempdir, nil
}
