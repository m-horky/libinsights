package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MatusOllah/slogcolor"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"

	. "github.com/RedHatInsights/rhc-insights"
)

func init() {
	CONFIGURATIONS_DIR = "./insights.d/"

	{
		// TODO Log into a file
		// Configure logging
		debug := false
		for _, arg := range os.Args {
			if arg == "--debug" {
				debug = true
				break
			}
		}

		opts := slogcolor.DefaultOptions
		opts.NoColor = !isatty.IsTerminal(os.Stderr.Fd())
		opts.LevelTags = map[slog.Level]string{
			slog.LevelDebug: color.New(color.FgYellow, color.Bold).Sprint("DEBUG"),
			slog.LevelInfo:  color.New(color.FgGreen, color.Bold).Sprint("INFO"),
			slog.LevelWarn:  color.New(color.FgHiRed, color.Bold).Sprint("WARN"),
			slog.LevelError: color.New(color.FgRed, color.Bold).Sprint("ERROR"),
		}
		opts.MsgPrefix = "> "
		if debug {
			opts.Level = slog.LevelDebug
			logger := slog.New(slogcolor.NewHandler(os.Stderr, opts))
			slog.SetDefault(logger)
		} else {
			opts.Level = slog.LevelError
			slog.SetDefault(slog.New(slogcolor.NewHandler(os.Stderr, opts)))
		}
	}

	{
		// Configure ingress
		// TODO Read rhsm.conf (or equivalent)
		if useStage := os.Getenv("_STAGE"); useStage != "" {
			slog.Debug("using stage Ingress")
			Ingress.URL.Host = "cert.console.stage.redhat.com:443"
		}
		_ = Ingress.SetCertAuth("/etc/pki/consumer/cert.pem", "/etc/pki/consumer/key.pem")
		slog.Debug("using certificate authorization")
	}

	{
		// Configure proxy
		// TODO Support stuff like HTTPS_PROXY, NO_PROXY
		if proxyURL := os.Getenv("HTTP_PROXY"); proxyURL != "" {
			proxy, err := url.Parse(proxyURL)
			if err != nil {
				slog.Error("could not parse proxy", "error", err.Error())
			} else {
				slog.Debug("using proxy", "url", proxy)
				Ingress.Proxy = proxy
			}
		}
	}
}

var ErrorNotImplemented = fmt.Errorf("not implemented")

func main() {
	// TODO Bash completion for collectors and flags
	cmd := &cli.Command{
		Name:            "rhc insights",
		HideHelpCommand: true,
		Usage:           "Collect and upload data",
		UsageText:       "rhc insights COMMAND [FLAGS]",
		Commands: []*cli.Command{
			{
				Name:      "run",
				Action:    doRun,
				Usage:     "run collector",
				UsageText: "rhc insights run [FLAGS] COLLECTOR",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "keep",
						Usage: "do not delete the data",
					},
					&cli.BoolFlag{
						Name:  "no-upload",
						Usage: "do not upload data",
					},
				},
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "collector", Min: 1, Max: 1},
				},
			},
			{
				Name:      "ls",
				Action:    doList,
				Usage:     "list collectors",
				UsageText: "rhc insights ls [FLAGS]",
			},
			{
				Name:      "ps",
				Action:    doTimers,
				Usage:     "list collector timers",
				UsageText: "rhc insights ps [FLAGS]",
			},
			{
				Name:      "enable",
				Action:    doEnable,
				Usage:     "enable collector timer",
				UsageText: "rhc insights enable [FLAGS] COLLECTOR",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "collector", Min: 1, Max: 1},
				},
			},
			{
				Name:      "disable",
				Action:    doDisable,
				Usage:     "disable collector timer",
				UsageText: "rhc insights disable [FLAGS] COLLECTOR",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "collector", Min: 1, Max: 1},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "format",
				Usage: "output format (options: 'json')",
				Validator: func(s string) error {
					switch s {
					case "":
						return nil
					case "json":
						return nil
					default:
						return fmt.Errorf("invalid format: %s", s)
					}
				},
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "enable debug logging",
			},
		},
	}

	slog.Info("starting", slog.String("args", strings.Join(os.Args, " ")))
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	slog.Info("done")
}

func doList(ctx context.Context, cmd *cli.Command) error {
	switch cmd.Value("format") {
	case "json":
		return ErrorNotImplemented
	default:
		return doListHuman(ctx, cmd)
	}
}

func doListHuman(ctx context.Context, cmd *cli.Command) error {
	collectors, err := GetCollectors()
	if err != nil {
		return err
	}

	// TODO Create a table with fields 'ID', 'Name', sorted by ID
	fmt.Println("ID NAME")
	for _, collector := range collectors {
		fmt.Println(collector.Meta.ID, collector.Meta.Name)
	}
	return nil
}

func doRun(ctx context.Context, cmd *cli.Command) error {
	switch cmd.Value("format") {
	case "json":
		return ErrorNotImplemented
	default:
		return doRunHuman(ctx, cmd)
	}
}

func doRunHuman(ctx context.Context, cmd *cli.Command) error {
	collector, err := GetCollector(cmd.StringArgs("collector")[0])
	if err != nil {
		return err
	}
	keep := cmd.Bool("keep") || cmd.Bool("no-upload")
	upload := !cmd.Bool("no-upload")

	// TODO Do not print temporary text if not in interactive console
	fmt.Printf("Executing '%s'\n", collector.Meta.Name)
	start := time.Now()
	tempdir, err := Collect(collector)
	delta := time.Since(start)
	fmt.Printf("\033[0K\r")

	defer func() {
		if keep {
			slog.Debug("keeping temporary directory", "path", tempdir)
			return
		}
		err = os.RemoveAll(tempdir)
		if err == nil {
			slog.Debug("wiped temporary directory", "path", tempdir)
		} else {
			slog.Warn("didn't wipe temporary directory", "path", tempdir, "err", err)
		}
	}()
	if err != nil {
		return err
	}

	if delta > time.Second {
		fmt.Printf("Execution of '%s' took %s.\n", collector.Meta.Name, delta.Truncate(time.Second/10))
	}
	if keep {
		fmt.Printf("Data have been kept in '%s'.\n", tempdir)
	}
	if !upload {
		slog.Debug("skipping data upload")
		return nil
	}

	archive, err := Compress(tempdir)
	if err != nil {
		return err
	}
	defer func() {
		err = os.Remove(archive)
		if err == nil {
			slog.Debug("wiped archive", "path", archive)
		} else {
			slog.Warn("did not wipe archive", "path", archive, "err", err)
		}
	}()

	return Upload(archive, collector.Exec.ContentType)
}

func doTimers(ctx context.Context, cmd *cli.Command) error {
	// TODO If we are not root, pass --user
	return ErrorNotImplemented
}

func doEnable(ctx context.Context, cmd *cli.Command) error {
	// TODO If we are not root, pass --user
	return ErrorNotImplemented
}

func doDisable(ctx context.Context, cmd *cli.Command) error {
	// TODO If we are not root, pass --user
	return ErrorNotImplemented
}
