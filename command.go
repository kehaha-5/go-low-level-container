package main

import (
	"fmt"
	"go-low-level-simple-runc/cgroups/limit"
	"go-low-level-simple-runc/container"
	"go-low-level-simple-runc/network"
	_ "go-low-level-simple-runc/nsenter"

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
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environment",
		},
		cli.StringFlag{
			Name:  "net",
			Usage: "set container network name",
		},
	},
	Action: func(c *cli.Context) error {
		if len(c.Args()) < 2 {
			return fmt.Errorf("miss exec cmd")
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
			EnvList:       c.StringSlice("e"),
			ImageName:     c.Args()[0],
			Net:           c.String("net"),
		}

		if runArgs.Tty && runArgs.Detach {
			return fmt.Errorf("it and d param can not work together")
		}

		if err := container.RunContainer(runArgs); err != nil {
			return fmt.Errorf("run container error %+v", err)
		}
		return nil
	},
}

var InitCmd = cli.Command{
	Name:  "init",
	Usage: "can not be useed outside",
	Action: func(c *cli.Context) error {
		if err := container.RunContainerProgram(); err != nil {
			return fmt.Errorf("init error %+v", err)
		}
		return nil
	},
}

var listContainer = cli.Command{
	Name:  "ps",
	Usage: "list all container",
	Action: func(c *cli.Context) error {
		slog.Info("ps command")
		files, err := os.ReadDir(container.GetConfigSavePath())
		if err != nil {
			return fmt.Errorf("read configfile error %v", err)
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
		return nil
	},
}

var logsContainer = cli.Command{
	Name:  "log",
	Usage: "show container log",
	Action: func(c *cli.Context) error {
		slog.Info("log command")
		if len(c.Args()) == 0 {
			slog.Error("too less args to run log command")
			return nil
		}
		containerName := c.Args()[0]
		log, err := container.GetLogByContainerName(containerName)
		if err != nil {
			slog.Error("GetLogByContainerName", "err", err)
		}
		fmt.Fprintln(os.Stdout, log)
		return nil
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
	Action: func(c *cli.Context) error {
		//This is for callback
		if os.Getenv(container.CONTAINERIDENV) != "" {
			slog.Info("exec", "pid callback pid", os.Getenv(container.CONTAINERIDENV))
			return nil
		}
		if len(c.Args()) < 2 {
			return fmt.Errorf("miss exec cmd")
		}
		containerName := c.Args()[0]
		containerCmd := c.Args()[1:]
		if err := container.Exce(containerName, containerCmd, c.Bool("it")); err != nil {
			return fmt.Errorf("exec err %v", err)
		}
		return nil
	},
}

var stopContainer = cli.Command{
	Name:  "stop",
	Usage: "stop container name",
	Action: func(c *cli.Context) error {
		if len(c.Args()) == 0 {
			return fmt.Errorf("less cmd too run")
		}
		containerName := c.Args()
		for _, itme := range containerName {
			err := container.StopContainerByName(itme)
			if err != nil {
				return fmt.Errorf("stop err %v", err)
			}
		}
		return nil
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
	Action: func(c *cli.Context) error {
		if len(c.Args()) == 0 {
			return fmt.Errorf("specify container name")
		}
		containerName := c.Args()
		for _, itme := range containerName {
			err := container.Rm(itme, c.Bool("f"))
			if err != nil {
				slog.Error("rm", "error", fmt.Errorf("name %s err %v", itme, err))
			}
		}
		return nil
	},
}

var commitContainer = cli.Command{
	Name:  "commit",
	Usage: "commit container name ",
	Action: func(c *cli.Context) error {
		if len(c.Args()) == 0 {
			return fmt.Errorf("specify container name")
		}
		if err := container.CommitContainer(c.Args().Get(0), c.Args().Get(1)); err != nil {
			return fmt.Errorf("commit %v", err)
		}
		return nil
	},
}

var networkCmd = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "d",
					Usage: "network driver",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				if err := network.Init(); err != nil {
					return fmt.Errorf("network init error: %+v", err)
				}
				err := network.CreateNetwork(context.String("d"), context.String("subnet"), context.Args()[0])
				if err != nil {
					return fmt.Errorf("create network error: %+v", err)
				}
				return nil
			},
		}, {
			Name:  "ls",
			Usage: "list all container network",
			Action: func(context *cli.Context) error {
				if err := network.Init(); err != nil {
					return fmt.Errorf("network init error: %+v", err)
				}
				w := tabwriter.NewWriter(os.Stdout, 12, 1, 5, ' ', tabwriter.TabIndent)
				fmt.Fprint(w, "ID\tNAME\tIpRange\tDriver\n")
				network.ShowAllNetworks(w)
				w.Flush()
				return nil
			},
		}, {
			Name:  "remove",
			Usage: "remove a container network name ...",
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				if err := network.Init(); err != nil {
					return fmt.Errorf("network init error: %+v", err)
				}
				for _, item := range context.Args() {
					if err := network.RemoveNetwork(item); err != nil {
						return err
					}
				}
				return nil
			},
		},
	},
}
