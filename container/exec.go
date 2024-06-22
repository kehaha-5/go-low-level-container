package container

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
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
	slog.Info("exec", "pid", pid)
	slog.Info("exec", "cmd", cmdStr)

	containerEnvs, err := getContainerEnvByPid(pid)
	if err != nil {
		return errors.WithStack(err)
	}
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err = cmd.Run(); err != nil {
		return err
	}
	return nil
}

func getContainerEnvByPid(pid string) ([]string, error) {
	envStr, err := os.ReadFile(path.Join(fmt.Sprintf("/proc/%s/environ", pid)))
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get pid %s environ", pid)
	}

	return strings.Split(string(envStr), "\u0000"), nil
}
