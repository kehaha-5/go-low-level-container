package main

import (
	"log/slog"
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
	},
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			slog.Error("args is too less to run")
			return
		}

		tty := c.Bool("it")

		if err := container.RunContainer(tty, c.Args()); err != nil {
			slog.Error("run container", err)
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
