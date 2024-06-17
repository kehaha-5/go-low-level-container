package container

import (
	"strconv"
	"syscall"
)

func StopContainerByName(name string) error {
	pid, err := getPidByContainerName(name)
	if err != nil {
		return err
	}
	intPid, err := strconv.Atoi(pid)
	if err != nil {
		return err
	}
	if err := syscall.Kill(intPid, syscall.SIGTERM); err != nil {
		return err
	}

	return modifyContainerStatusToStopByName(name)
}
