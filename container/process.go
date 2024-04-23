package container

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"simple-docker/common"
	"strings"
	"syscall"
)

const (
	readonlyLayerFilename string = "readOnly"
	wirteLayerFilename    string = "wirteOnly"
	workLayerFilename     string = "work"
)

type workSpace struct {
	readonlyLayer string
	wirteLayer    string
	workLayer     string
	mountRoot     string
	volumeRoot    []string
}

var workSpaceInfo workSpace

// 初始化容器进程
func initContainerParent(tty bool, volumeArg string, containerId string) (*exec.Cmd, *os.File, error) {
	readPipe, writePipe, err := newPipe()
	if err != nil {
		slog.Error("new pipe", err)
		return nil, nil, err
	}
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}

	if tty {
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
	}

	// 在进程启动前overlay创建工作区 保证在后面挂载了root后对文件进行修改 不会破坏原始镜像
	// pwd, err := os.Getwd()
	pwd := "/workspaces/go-simple-docker/runEnv"
	// if err != nil {
	// 	return nil, nil, fmt.Errorf("get exceutable path  %v ", err)
	// }
	//解析volume
	workSpaceInfo.volumeRoot = volumeUrlExtract(volumeArg)

	if err := newWorkSpace(pwd, "busybox.tar", path.Join("/mnt", containerId)); err != nil {
		return nil, nil, fmt.Errorf("new work space %v", err)
	}

	cmd.ExtraFiles = []*os.File{readPipe}
	cmd.Env = append(cmd.Environ(), "mountRoot="+workSpaceInfo.mountRoot)

	return cmd, writePipe, nil
}

func newPipe() (r *os.File, w *os.File, err error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

// 初始化工作区 并挂载overlay
// root 镜像的根目录 baseImg 镜像名称 mnt overlay挂载点
func newWorkSpace(root string, baseImg string, mnt string) error {
	workSpaceInfo.readonlyLayer = path.Join(root, readonlyLayerFilename)
	workSpaceInfo.wirteLayer = path.Join(root, wirteLayerFilename)
	workSpaceInfo.workLayer = path.Join(root, workLayerFilename)
	workSpaceInfo.mountRoot = path.Join(root, mnt)
	if err := createOnlyReadLayer(root, baseImg); err != nil {
		return err
	}
	if err := createLayer(workSpaceInfo.wirteLayer); err != nil {
		return err
	}
	if err := createLayer(workSpaceInfo.workLayer); err != nil {
		return err
	}
	if err := createOverlay(); err != nil {
		return err
	}

	if len(workSpaceInfo.volumeRoot) != 0 {
		if len(workSpaceInfo.volumeRoot) < 1 {
			slog.Error("volume params not correct.")
		} else {
			if err := mountVolume(workSpaceInfo.volumeRoot); err != nil {
				slog.Error("mountVolume", "err", err)
			}
		}
	}

	return nil
}

// 创建overlay中的只读层，一般从基础镜像中进行解压
func createOnlyReadLayer(root string, baseImg string) error {
	// 确定基础镜像是否存在
	baseImgPath := path.Join(root, baseImg)
	if !common.FileExist(baseImgPath) {
		return fmt.Errorf("baseimg %s not exists", baseImgPath)
	}

	// 创建readonly层
	exist, err := common.PathExist(workSpaceInfo.readonlyLayer)
	if err != nil {
		return fmt.Errorf("fail to judge whether readonly dir exists. %v", err)
	}
	if !exist {
		if err = os.Mkdir(workSpaceInfo.readonlyLayer, 0777); err != nil {
			return fmt.Errorf("fail to mkidr readonly dir %s exists. %v", workSpaceInfo.readonlyLayer, err)
		}
	}

	// 解压基础镜像到readonly层
	if _, err = exec.Command("tar", "-xvf", baseImgPath, "-C", workSpaceInfo.readonlyLayer).CombinedOutput(); err != nil {
		return fmt.Errorf("untar the img %s to %s  %v", baseImgPath, workSpaceInfo.readonlyLayer, err)
	}

	return nil
}

// 创建工作层
func createLayer(root string) error {
	if err := os.Mkdir(root, 0777); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("mkdir layer %s  %v", root, err)
		}
	}
	return nil
}

// 创建overlay挂载文件 并进行overlay挂载
func createOverlay() error {
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
func mountVolume(volumeUrl []string) error {
	// 创建宿主机文件
	parantUrl := volumeUrl[0]
	if err := os.MkdirAll(parantUrl, 0777); err != nil {
		return err
	}

	// 创建容器挂载点 在挂载点中创建
	containerUrl := path.Join(workSpaceInfo.mountRoot, volumeUrl[1])
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

func delWorkSpace() error {
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

	workDelErr := os.RemoveAll(workSpaceInfo.wirteLayer)
	moutnDelErr := os.RemoveAll(workSpaceInfo.mountRoot)

	if workDelErr != nil {
		return workDelErr
	}
	if moutnDelErr != nil {
		return moutnDelErr
	}
	return nil
}

func volumeUrlExtract(volumeUrl string) []string {
	if volumeUrl == "" {
		return nil
	}
	return strings.Split(volumeUrl, ":")
}
