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
	Tty          bool
	VolumeArg    string
	LimitResConf *limit.ResourceConfig
	Args         []string
}

func RunContainer(args *RunCommandArgs) error {
	cmd, writePipe, err := initContainerParent(args.Tty, args.VolumeArg)
	if err != nil {
		return err
	}

	slog.Info("create container process and running ")
	if err := cmd.Start(); err != nil {
		return err
	}

	slog.Info("limit rescoure", "mem", args.LimitResConf.Memory, "cpu", args.LimitResConf.Cpu, "cpuset", args.LimitResConf.Cpuset)
	cg := cgroups.NewCgroupManager("test_cgroup")

	if err := cg.Set(args.LimitResConf); err == nil {
		defer cg.Destroy()
		if err := cg.Apply(cmd.Process.Pid); err != nil {
			return err
		}
	} else {
		return err
	}

	sendMsgToPipe(writePipe, args.Args)

	cmd.Wait()

	if err := delWorkSpace(); err != nil {
		return fmt.Errorf("delWorkSpace %v", err)
	}
	os.Exit(1)
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
