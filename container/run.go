package container

import (
	"fmt"
	"go-low-level-simple-runc/cgroups"
	"go-low-level-simple-runc/cgroups/limit"
	"log/slog"
	"os"
	"strings"
)

type RunCommandArgs struct {
	Tty           bool
	VolumeArg     string
	LimitResConf  *limit.ResourceConfig
	CommandArgs   []string
	Detach        bool
	ContainerName string
	ImageName     string
}

func RunContainer(args *RunCommandArgs) error {
	var containerInfo ContainerInfos
	containerInfo.SetContainerName(args.ContainerName)
	cmd, writePipe, workSpace, err := initContainerParent(args.Tty, args.VolumeArg, containerInfo.Name, args.ImageName)
	if err != nil {
		return err
	}

	slog.Info("create container process and running ")
	if err := cmd.Start(); err != nil {
		return err
	}

	// 记录container信息
	containerName, err := containerInfo.RecordContainerInfo(cmd.Process.Pid, args.CommandArgs, args.VolumeArg)
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
		if err := workSpace.delWorkSpace(); err != nil {
			slog.Error("delWorkSpace", "err", err)
		}
		containerInfo.DeleteContainerInfo()
		defer cg.Destroy()
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, containerInfo.Name)
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
