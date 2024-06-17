package main

import (
	"fmt"
	"go-low-level-simple-runc/cgroups/limit"
	"go-low-level-simple-runc/container"
	_ "go-low-level-simple-runc/container/nsenter"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli"
)

var RunCmd = cli.Command{
	Name:  "run",
	Usage: "run [Option] image [Command] [Args]",
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
			slog.Error("miss exec cmd")
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
			CommandArgs:   c.Args()[1:],
			Detach:        c.Bool("d"),
			ContainerName: c.String("name"),
			ImageName:     c.Args()[0],
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

var logsContainer = cli.Command{
	Name:  "log",
	Usage: "show container log",
	Action: func(c *cli.Context) {
		slog.Info("log command")
		if len(c.Args()) == 0 {
			slog.Error("too less args to run log command")
			return
		}
		containerName := c.Args()[0]
		log, err := container.GetLogByContainerName(containerName)
		if err != nil {
			slog.Error("GetLogByContainerName", "err", err)
			return
		}
		fmt.Fprintln(os.Stdout, log)

	},
}

var execContainer = cli.Command{
	Name:  "exec",
	Usage: "Execute a command in a running container",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: "Keep STDIN open even if not attached and Allocate a pseudo-TTY",
		},
	},
	Action: func(c *cli.Context) {
		if len(c.Args()) < 2 {
			slog.Error("miss exec cmd")
			return
		}
		containerName := c.Args()[0]
		containerCmd := c.Args()[1:]
		if err := container.Exce(containerName, containerCmd, c.Bool("it")); err != nil {
			slog.Error("exec", "err", err)
		}
	},
}

var stopContainer = cli.Command{
	Name:  "stop",
	Usage: "stop container name",
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			slog.Error("less cmd too run")
			return
		}
		containerName := c.Args()
		for _, itme := range containerName {
			err := container.StopContainerByName(itme)
			if err != nil {
				slog.Error("stop", "err", err)
			}
		}
	},
}

var rmContainer = cli.Command{
	Name:  "rm",
	Usage: "rm container name",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "f",
			Usage: "force rm",
		},
	},
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			slog.Error("specify container name")
			return
		}
		containerName := c.Args()
		for _, itme := range containerName {
			err := container.Rm(itme, c.Bool("f"))
			if err != nil {
				slog.Error("rm", "err", err)
			}
		}

	},
}

var commitContainer = cli.Command{
	Name:  "commit",
	Usage: "commit container name ",
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			slog.Error("specify container name ")
			return
		}
		if err := container.CommitContainer(c.Args().Get(0), c.Args().Get(1)); err != nil {
			slog.Error("commit", "err", err)
		}
	},
}
