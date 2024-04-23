package main

import (
	"fmt"
	"log/slog"
	"os"
	"simple-docker/cgroups/limit"
	"simple-docker/container"
	"text/tabwriter"

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
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.IntFlag{
			Name:  "cpushare",
			Usage: "CPU shares (relative weight)",
		},
		cli.IntFlag{
			Name:  "cpusset",
			Usage: "CPUs in which to allow execution",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "Memory limit ",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "Bind mount a volume",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
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
			CommandArgs:   c.Args(),
			Detach:        c.Bool("d"),
			ContainerName: c.String("name"),
		}

		if runArgs.Tty && runArgs.Detach {
			slog.Error("it and d param can not work together")
			return
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

var listContainer = cli.Command{
	Name:  "ps",
	Usage: "list all container",
	Action: func(c *cli.Context) {
		slog.Info("ps command")
		files, err := os.ReadDir(container.GetConfigSavePath())
		if err != nil {
			slog.Error("read configfile", "err", err)
		}
		w := tabwriter.NewWriter(os.Stdout, 12, 1, 5, ' ', tabwriter.TabIndent)
		fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")

		for _, file := range files {
			var info container.ContainerInfos
			if err := container.GetInfoByContainerName(file.Name(), &info); err != nil {
				slog.Error("GetInfoByContainerName", "err", err)
				continue
			}
			info.WirteInfoToTabwriter(w)
		}
		w.Flush()
	},
}
