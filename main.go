package main

import (
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
	}

	app.Run(os.Args)

	// setLogConf()

}
