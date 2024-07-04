package container

import (
	"github.com/pkg/errors"
)

func RestartContainer(name string) error {
	info := ContainerInfos{}
	if err := GetInfoByContainerName(name, &info); err != nil {
		return errors.Wrap(err, "fail to find the container info")
	}

	if err := StopContainerByName(info.Name); err != nil {
		return errors.Wrap(err, "fail to stop container")
	}


	return StartContainerByName(name)
}
