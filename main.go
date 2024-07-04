package main

import (
	"log/slog"
	"os"

	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "simple-docker"

	app.Commands = []cli.Command{
		RunCmd,
		InitCmd,
		listContainer,
		logsContainer,
		execContainer,
		stopContainer,
		rmContainer,
		commitContainer,
		networkCmd,
		startCmd,
		restartCmd,
	}

	app.Before = func(context *cli.Context) error {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		slog.Error(err.Error())
	}

	// setLogConf()

}
