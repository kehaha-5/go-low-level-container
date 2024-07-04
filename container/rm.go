package container

import (
	"fmt"

	"github.com/kehaha-5/go-low-level-container/network"
	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
)

func Rm(name string, force bool) error {
	data := ContainerInfos{}
	err := GetInfoByContainerName(name, &data)
	if err != nil {
		return errors.Wrap(err, "fail to get info by conatiner name")
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
	if data.IpInfo.ID != "" {
		if err := network.DisConnect(&data.IpInfo); err != nil {
			return fmt.Errorf("fail to disconnect error %v", err)
		}
	}

	if err := netns.DeleteNamed(data.Name); err != nil {
		return fmt.Errorf("fail to delete netns %v", err)
	}

	if err := data.del(); err != nil {
		return err
	}

	return workSpaceInfo.delWorkSpace()
}
