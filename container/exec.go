package container

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

func Exce(name string, cmdArr []string, tty bool) error {
	pid, err := getPidByContainerName(name)
	if err != nil {
		return err
	}

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if tty {
		cmd.Stdin = os.Stdin
	}

	if err = os.Setenv(CONTAINERIDENV, pid); err != nil {
		return err
	}

	cmdStr := strings.Join(cmdArr[0:], " ")
	if err = os.Setenv(CONTAINERCMDENV, cmdStr); err != nil {
		return err
	}
	slog.Info("exec","pid", pid)
	slog.Info("exec","cmd", cmdStr)

	if err = cmd.Run(); err != nil {
		return err
	}
	return nil
}
