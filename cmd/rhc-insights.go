package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	. "github.com/RedHatInsights/libinsights"
)

func init() {
	CONFIGURATIONS_DIR = "./insights.d/"

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
}

var ErrorNotImplemented = fmt.Errorf("not implemented")

func main() {
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
				UsageText: "rhc insights enable [FLAGS]",
			},
			{
				Name:      "disable",
				Action:    doDisable,
				Usage:     "disable collector timer",
				UsageText: "rhc insights disable [FLAGS]",
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
	collector, err := GetCollector(cmd.Args().Get(0))
	if err != nil {
		return err
	}
	keep := cmd.Bool("keep") || cmd.Bool("no-upload")
	upload := !cmd.Bool("no-upload")

	fmt.Printf("executing '%s'", collector.Meta.Name)
	fmt.Printf("\033[0K\r")

	tempdir, err := Collect(collector)
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

	// TODO Print how long it took.
	if keep {
		fmt.Printf("Data have been kept in '%s'.\n", tempdir)
	}
	if !upload {
		slog.Debug("skipping data upload")
		return nil
	}

	// TODO Upload
	return nil
}

func doTimers(ctx context.Context, cmd *cli.Command) error {
	// TODO If we are not root, pass --user
	return nil
}

func doEnable(ctx context.Context, cmd *cli.Command) error {
	return nil
}

func doDisable(ctx context.Context, cmd *cli.Command) error {
	return nil
}
