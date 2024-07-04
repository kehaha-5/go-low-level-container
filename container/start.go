package container

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/kehaha-5/go-low-level-container/network"
	"github.com/pkg/errors"
)

func StartContainerByName(name string) error {
	info := ContainerInfos{}
	if err := GetInfoByContainerName(name, &info); err != nil {
		return errors.Wrap(err, "fail to get container info")
	}

	if info.IpInfo.ID != "" {
		if err := network.ConfigMapping(&info.IpInfo); err != nil {
			return errors.Wrap(err, "fail to config mapping")
		}
	}
	readPipe, writePipe, cmd, err := initContainerParent()
	if err != nil {
		return errors.Wrap(err, "fail to init container parent")
	}

	delLogByContainerName(info.Name)

	lopfile, err := createlogfilePointer(info.Name)
	if err != nil {
		return errors.Wrap(err, "fail to get log file ptr")
	}
	cmd.Stdout = lopfile
	cmd.Stderr = lopfile

	setProcessEnv(cmd, readPipe, info.Env)

	initArgs := &initArgs{
		Hostname:  info.Name,
		MountRoot: getMountRootPathByContainerName(info.Name),
		Args:      strings.Split(info.Command, " "),
		NetnsName: info.Name,
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	info.UpdatePid(cmd.Process.Pid)
	if info.Cg.Path != "" {
		slog.Debug("set cg")
		if err := info.Cg.Apply(cmd.Process.Pid); err != nil {
			slog.Error("set cg", "err", err)

		}
	}

	if err := sendMsgToPipe(writePipe, initArgs); err != nil {
		return errors.WithStack(err)
	}
	// 记录container信息
	if err := info.modifyContainerStatusByName(Running); err != nil {
		return fmt.Errorf("recordContainerInfo %+v", err)
	}

	return nil
}
