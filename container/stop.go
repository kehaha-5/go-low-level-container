package container

import (
	"log/slog"
	"strconv"
	"strings"
	"syscall"

	"github.com/kehaha-5/go-low-level-container/network"
)

func StopContainerByName(name string) error {
	info := ContainerInfos{}
	err := GetInfoByContainerName(name, &info)
	if err != nil {
		return err
	}
	intPid, err := strconv.Atoi(info.Pid)
	if err != nil {
		return err
	}
	if err := syscall.Kill(intPid, syscall.SIGTERM); err != nil && !strings.Contains(err.Error(), "no such process") {
		slog.Error("stop", "kill pid", err)
	}

	if info.IpInfo.ID != "" {
		if err := network.DelIptRules(&info.IpInfo); err != nil {
			slog.Error("stop", "del ipt rules", err)
		}
	}

	return info.modifyContainerStatusByName(Stop)
}
