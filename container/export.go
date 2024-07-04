package container

import (
	"fmt"
	"os/exec"
)

func ExportCommitContainer(name string, imgetar string) error {
	info := ContainerInfos{}
	if err := GetInfoByContainerName(name, &info); err != nil {
		return err
	}
	workSpace := getWorkSpackInfoByContainerInfos(&info)

	imgetar += ".tar"
	if _, err := exec.Command("tar", "-czf", imgetar, "-C", workSpace.mountRoot, ".").CombinedOutput(); err != nil {
		return fmt.Errorf("image tar error %v", err)
	}
	return nil
}
