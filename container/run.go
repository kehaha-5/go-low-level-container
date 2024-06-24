package container

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kehaha-5/go-low-level-simple-docker/cgroups"
	"github.com/kehaha-5/go-low-level-simple-docker/cgroups/limit"
	"github.com/kehaha-5/go-low-level-simple-docker/network"

	"github.com/pkg/errors"
)

type RunCommandArgs struct {
	Tty           bool
	VolumeArg     string
	LimitResConf  *limit.ResourceConfig
	CommandArgs   []string
	Detach        bool
	ContainerName string
	ImageName     string
	EnvList       []string
	Net           string
}

func RunContainer(args *RunCommandArgs) error {
	containerInfo := &ContainerInfos{}
	containerInfo.SetContainerName(args.ContainerName)
	cmd, writePipe, workSpace, err := initContainerParent(args.Tty, args.VolumeArg, containerInfo.Name, args.ImageName, args.EnvList)
	if err != nil {
		return errors.WithStack(err)
	}

	slog.Info("create container process and running ")
	if err := cmd.Start(); err != nil {
		return err
	}

	// 记录container信息
	containerName, err := containerInfo.RecordContainerInfo(cmd.Process.Pid, args.CommandArgs, args.VolumeArg)
	if err != nil {
		return fmt.Errorf("recordContainerInfo %+v", err)
	}

	slog.Info("limit rescoure", "mem", args.LimitResConf.Memory, "cpu", args.LimitResConf.Cpu, "cpuset", args.LimitResConf.Cpuset)
	cg := cgroups.NewCgroupManager(containerName)
	if err := cg.Set(args.LimitResConf); err == nil {
		if err := cg.Apply(cmd.Process.Pid); err != nil {
			slog.Error("set cg", "err", err)
		}
	} else {
		slog.Error("set cg", "err", err)
	}

	slog.Info("save contianer info")

	sendMsgToPipe(writePipe, args.CommandArgs)

	if args.Net != "" {
		if err := network.Init(); err != nil {
			return errors.WithStack(err)
		}
		if err := network.Connect(args.Net, containerInfo.Id, containerInfo.Pid, containerInfo.PortMapping); err != nil {
			return errors.Wrap(err, "fail to connect net")
		}
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

func sendMsgToPipe(writePipe *os.File, args []string) {
	slog.Info("send msg to pipe", "args", args)
	writePipe.WriteString(strings.Join(args, " "))
	writePipe.Close()
}
