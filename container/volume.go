package container

import (
	"fmt"
	"go-low-level-simple-runc/common"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
)

const (
	defaultRoot          string = "container"
	defaultReadonlyLayer string = "readOnly"
	defaultWirteLayer    string = "wirteOnly"
	defaultWorkLayer     string = "work"
	defaultImagesPath    string = "images"
	defaultMntRoot       string = "mnt"
	root                 string = "/workspaces/go-low-level-simple-runc/runEnv"
)

type workSpace struct {
	containerName string
	readonlyLayer string
	wirteLayer    string
	workLayer     string
	mountRoot     string
	volumeRoot    []string
}

// 初始化工作区 并挂载overlay
// root 镜像的根目录 baseImg 镜像名称 mnt overlay挂载点
func NewWorkSpace(baseImgName string, containerName string, volumeArg string) (workSpace, error) {
	workSpaceInfo := workSpace{}
	workSpaceInfo.readonlyLayer = path.Join(root, defaultReadonlyLayer, baseImgName)
	workSpaceInfo.wirteLayer = path.Join(root, defaultRoot, containerName, defaultWirteLayer)
	workSpaceInfo.workLayer = path.Join(root, defaultRoot, containerName, defaultWorkLayer)
	workSpaceInfo.mountRoot = path.Join(root, defaultRoot, containerName, defaultMntRoot)
	workSpaceInfo.containerName = containerName
	workSpaceInfo.volumeRoot = volumeUrlExtract(volumeArg)

	if err := workSpaceInfo.createReadOnlyLayer(root, baseImgName); err != nil {
		return workSpace{}, err
	}
	if err := createLayer(workSpaceInfo.wirteLayer); err != nil {
		return workSpace{}, err
	}
	if err := createLayer(workSpaceInfo.workLayer); err != nil {
		return workSpace{}, err
	}
	if err := workSpaceInfo.createOverlay(); err != nil {
		return workSpace{}, err
	}

	if len(workSpaceInfo.volumeRoot) != 0 {
		if len(workSpaceInfo.volumeRoot) < 1 {
			slog.Error("volume params not correct.")
		} else {
			if err := workSpaceInfo.mountVolume(); err != nil {
				slog.Error("mountVolume", "err", err)
			}
		}
	}

	return workSpaceInfo, nil
}

// 创建overlay中的只读层，一般从基础镜像中进行解压
func (workSpaceInfo *workSpace) createReadOnlyLayer(root string, baseImgName string) error {
	// 确定基础镜像是否存在
	baseImgPath := path.Join(root, defaultImagesPath, baseImgName)
	baseImgPath += ".tar"
	if !common.FileExist(baseImgPath) {
		return fmt.Errorf("baseimg %s not exists", baseImgPath)
	}

	// 创建readonly层
	exist, err := common.PathExist(workSpaceInfo.readonlyLayer)
	if err != nil {
		return errors.Wrap(err, "createReadOnlyLayer fail to judge whether readonly dir exists.")
	}
	if !exist {
		if err = os.MkdirAll(workSpaceInfo.readonlyLayer, 0777); err != nil {
			return errors.Wrap(err, "createReadOnlyLayer  mkdirall")
		}
	}

	// 解压基础镜像到readonly层
	if _, err = exec.Command("tar", "-xvf", baseImgPath, "-C", workSpaceInfo.readonlyLayer).CombinedOutput(); err != nil {
		return errors.Wrapf(err, "createReadOnlyLayer  untar the img %s to %s ", baseImgPath, workSpaceInfo.readonlyLayer)
	}
	return nil
}

// 创建overlay挂载文件 并进行overlay挂载
func (workSpaceInfo *workSpace) createOverlay() error {
	if err := os.MkdirAll(workSpaceInfo.mountRoot, 0777); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("mkdir mnt %s err %v", workSpaceInfo.mountRoot, err)
		}
	}

	//  mount -t overlay overlay -o lowerdir=A:B,upperdir=C,workdir=worker /tmp/test/merged
	// lowerdir 为只读层 upperdir 为读写层 work为工作层
	args := "lowerdir=" + workSpaceInfo.readonlyLayer + ",upperdir=" + workSpaceInfo.wirteLayer + ",workdir=" + workSpaceInfo.workLayer
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", args, workSpaceInfo.mountRoot)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mount overlay %v", err)
	}
	return nil
}

// 挂载volume层
func (workSpaceInfo *workSpace) mountVolume() error {
	// 创建宿主机文件
	parantUrl := workSpaceInfo.volumeRoot[0]
	if err := os.MkdirAll(parantUrl, 0777); err != nil {
		return err
	}

	// 创建容器挂载点 在挂载点中创建
	containerUrl := path.Join(workSpaceInfo.mountRoot, workSpaceInfo.volumeRoot[1])
	if err := os.MkdirAll(path.Join(containerUrl), 0777); err != nil {
		return err
	}

	// 挂载宿主机到容器
	cmd := exec.Command("mount", "--bind", parantUrl, containerUrl)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (workSpaceInfo *workSpace) delWorkSpace() error {
	// 卸载容器volume
	if len(workSpaceInfo.volumeRoot) == 2 {
		cmd := exec.Command("umount", path.Join(workSpaceInfo.mountRoot, workSpaceInfo.volumeRoot[1]))
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			slog.Error("volume ", "umount", err)
		}
	}

	// 卸载容器挂载
	cmd := exec.Command("umount", workSpaceInfo.mountRoot)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mountRoot umount %v ", err)
	}

	return os.RemoveAll(path.Join(root, defaultRoot, workSpaceInfo.containerName))

}

func getWorkSpackInfoByContainerInfos(info *ContainerInfos) workSpace {
	workSpaceInfo := workSpace{}
	workSpaceInfo.readonlyLayer = path.Join(root, defaultReadonlyLayer)
	workSpaceInfo.wirteLayer = path.Join(root, defaultRoot, info.Name, defaultWirteLayer)
	workSpaceInfo.workLayer = path.Join(root, defaultRoot, info.Name, defaultWorkLayer)
	workSpaceInfo.mountRoot = path.Join(root, defaultRoot, info.Name, defaultMntRoot)
	workSpaceInfo.volumeRoot = volumeUrlExtract(info.Volume)
	workSpaceInfo.containerName = info.Name
	return workSpaceInfo
}

func volumeUrlExtract(volumeUrl string) []string {
	if volumeUrl == "" {
		return nil
	}
	return strings.Split(volumeUrl, ":")
}

// 创建工作层
func createLayer(root string) error {
	if err := os.MkdirAll(root, 0777); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}
