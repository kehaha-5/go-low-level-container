package main

import (
	"log/slog"
	"simple-docker/cgroups/limit"
	"simple-docker/container"

	"github.com/urfave/cli"
)

var RunCmd = cli.Command{
	Name:  "run",
	Usage: "run [Command]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: "Keep STDIN open even if not attached and Allocate a pseudo-TTY",
		},
		cli.IntFlag{
			Name:  "c",
			Usage: "CPU shares (relative weight)",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "Memory limit ",
		},
		cli.IntFlag{
			Name:  "cs-c",
			Usage: "CPUs in which to allow execution",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "Bind mount a volume",
		},
	},
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			slog.Error("args is too less to run")
			return
		}

		var runArgs = &container.RunCommandArgs{
			Tty:       c.Bool("it"),
			VolumeArg: c.String("v"),
			LimitResConf: &limit.ResourceConfig{
				Cpu:    c.Int("c"),
				Cpuset: c.Int("cs-c"),
				Memory: c.String("m"),
			},
			Args: c.Args(),
		}

		if err := container.RunContainer(runArgs); err != nil {
			slog.Error("run container", "err", err)
		}
	},
}

var InitCmd = cli.Command{
	Name:  "init",
	Usage: "can not be useed outside",
	Action: func(c *cli.Context) {
		slog.Info("init come on")
		container.RunContainerProgram()
	},
}
