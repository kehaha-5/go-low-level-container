package container

import (
	"fmt"
	"log/slog"
	"os"
	"simple-docker/cgroups"
	"simple-docker/cgroups/limit"
	"strings"
)

type RunCommandArgs struct {
	Tty           bool
	VolumeArg     string
	LimitResConf  *limit.ResourceConfig
	CommandArgs   []string
	Detach        bool
	ContainerName string
}

func RunContainer(args *RunCommandArgs) error {
	// 生成容器id
	var containerInfo ContainerInfos
	containerInfo.RandomContainerId(10)

	cmd, writePipe, err := initContainerParent(args.Tty, args.VolumeArg, containerInfo.Id)
	if err != nil {
		return err
	}

	slog.Info("create container process and running ")
	if err := cmd.Start(); err != nil {
		return err
	}

	// 记录container信息
	containerName, err := containerInfo.RecordContainerInfo(cmd.Process.Pid, args.ContainerName, args.CommandArgs)
	if err != nil {
		return fmt.Errorf("recordContainerInfo %v", err)
	}

	slog.Info("limit rescoure", "mem", args.LimitResConf.Memory, "cpu", args.LimitResConf.Cpu, "cpuset", args.LimitResConf.Cpuset)
	cg := cgroups.NewCgroupManager(containerName)

	if err := cg.Set(args.LimitResConf); err == nil {

		if err := cg.Apply(cmd.Process.Pid); err != nil {
			return err
		}
	} else {
		return err
	}

	slog.Info("save contianer info")

	sendMsgToPipe(writePipe, args.CommandArgs)

	if args.Tty {
		cmd.Wait()
		if err := delWorkSpace(); err != nil {
			slog.Error("delWorkSpace", "err", err)
		}
		containerInfo.DeleteContainerInfo()
		defer cg.Destroy()
		os.Exit(1)
	}
	return nil
}

func RunContainerProgram() {
	if err := runContainerProgram(); err != nil {
		slog.Error("runContainerProgram", "err", err)
	}
}

func sendMsgToPipe(writePipe *os.File, args []string) {
	slog.Info("send msg to pipe", "args", args)
	writePipe.WriteString(strings.Join(args, " "))
	writePipe.Close()
}
