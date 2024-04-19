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
	}

	app.Run(os.Args)

	// setLogConf()

}

func setLogConf() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
}
