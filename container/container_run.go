package container

import (
	"log/slog"
	"os"
	"strings"
)

func RunContainer(tty bool,  args []string) error {
	cmd, writePipe, err := initContainer(tty)
	if err != nil {
		slog.Error("initContainer", err)
	}
	slog.Info("create container process and running ")
	if err := cmd.Start(); err != nil {
		slog.Error("cmd start", err)
		return err
	}
	sendMsgToPipe(writePipe, args)
	cmd.Wait()
	return nil
}

func RunContainerProgram() {
	if err := runContainerProgram(); err != nil {
		slog.Error("runContainerProgram", err)
	}
}

func sendMsgToPipe(writePipe *os.File, args []string) {
	slog.Info("send msg to pipe", "args", args)
	writePipe.WriteString(strings.Join(args, " "))
	writePipe.Close()
}
