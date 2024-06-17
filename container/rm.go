package container

import (
	"fmt"
)

func Rm(name string, force bool) error {
	data := ContainerInfos{}
	err := GetInfoByContainerName(name, &data)
	if err != nil {
		return err
	}

	workSpaceInfo := getWorkSpackInfoByContainerInfos(&data)
	if data.Status != Stop {
		if !force {
			return fmt.Errorf("container is running")
		}
	}

	if data.Status != Stop {
		if err := StopContainerByName(name); err != nil {
			return err
		}
	}

	if err := data.del(); err != nil {
		return err
	}

	return workSpaceInfo.delWorkSpace()
}
