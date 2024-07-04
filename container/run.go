package container

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/kehaha-5/go-low-level-container/cgroups"
	"github.com/kehaha-5/go-low-level-container/cgroups/limit"
	"github.com/kehaha-5/go-low-level-container/network"

	"github.com/pkg/errors"
)

type RunCommandArgs struct {
	Tty           bool
	VolumeArg     []string
	LimitResConf  *limit.ResourceConfig
	CommandArgs   []string
	Detach        bool
	ContainerName string
	ImageName     string
	EnvList       []string
	Net           string
	PortMapping   string
}

func RunContainer(args *RunCommandArgs) error {
	containerInfo := &ContainerInfos{}
	containerInfo.SetContainerName(args.ContainerName)
	cmd, writePipe, workSpace, err := initContainerParentWithNewWorkSpace(args.Tty, args.VolumeArg, containerInfo.Name, args.ImageName, args.EnvList)
	if err != nil {
		return errors.WithStack(err)
	}

	// 添加新的net namespace
	if err := exec.Command("ip", "netns", "add", containerInfo.Name).Run(); err != nil {
		return errors.Wrapf(err, "fail to add ip netns %s", containerInfo.Name)
	}

	initArgs := &initArgs{
		Hostname:  containerInfo.Name,
		MountRoot: workSpace.mountRoot,
		Args:      args.CommandArgs,
		NetnsName: containerInfo.Name,
	}

	slog.Info("create container process and running ")
	if err := cmd.Start(); err != nil {
		return err
	}

	containerInfo.setBaseInfo(cmd.Process.Pid, args)
	slog.Info("limit rescoure", "mem", args.LimitResConf.Memory, "cpu", args.LimitResConf.Cpu, "cpuset", args.LimitResConf.Cpuset)
	cg := cgroups.NewCgroupManager(containerInfo.Name)
	if err := cg.Set(args.LimitResConf); err == nil {
		if err := cg.Apply(cmd.Process.Pid); err != nil {
			slog.Error("set cg", "err", err)
		}
		containerInfo.SetCg(cg)
	} else {
		slog.Error("set cg", "err", err)
	}

	slog.Info("save contianer info")

	if err := sendMsgToPipe(writePipe, initArgs); err != nil {
		return errors.WithStack(err)
	}

	if args.Net != "" {
		if err := network.Init(); err != nil {
			return errors.WithStack(err)
		}
		ep, err := network.Connect(args.Net, containerInfo.Id, containerInfo.Name, containerInfo.PortMapping)
		if err != nil {
			return errors.Wrap(err, "fail to connect net")
		}
		containerInfo.SetNetInfo(ep)
	}

	// 记录container信息
	if err := containerInfo.RecordContainerInfo(); err != nil {
		return fmt.Errorf("recordContainerInfo %+v", err)
	}

	if args.Tty {
		cmd.Wait()
		if err := workSpace.delWorkSpace(); err != nil {
			slog.Error(fmt.Errorf("delWorkSpace err %+v", err).Error())
		}
		containerInfo.DeleteContainerInfo()
		defer cg.Destroy()
		return nil
	}
	fmt.Fprintln(os.Stdout, containerInfo.Name)
	return nil
}

func RunContainerProgram() error {
	return runContainerProgram()
}

func sendMsgToPipe(writePipe *os.File, args *initArgs) error {
	slog.Info("send msg to pipe", "args", args)
	jsonStr, err := json.Marshal(args)
	if err != nil {
		return errors.WithStack(err)
	}
	writePipe.WriteString(string(jsonStr))
	writePipe.Close()
	return nil
}
