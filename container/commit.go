package container

import (
	"fmt"
	"os/exec"
	"path"
)

const (
	defaultImagetar string = "images"
)

func CommitContainer(name string, imgetar string) error {
	info := ContainerInfos{}
	if err := GetInfoByContainerName(name, &info); err != nil {
		return err
	}
	workSpace := getWorkSpackInfoByContainerInfos(&info)

	if imgetar == "" {
		imgetar = path.Join(root, defaultImagetar, name)
	}
	imgetar += ".tar"
	if _, err := exec.Command("tar", "-czf", imgetar, "-C", workSpace.mountRoot, ".").CombinedOutput(); err != nil {
		return fmt.Errorf("image tar error %v", err)
	}
	return nil
}
