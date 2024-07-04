package container

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

type initArgs struct {
	Args      []string
	MountRoot string
	Hostname  string
	NetnsName string
}

// 执行容器内应用进程
// 挂载/proc
func runContainerProgram() error {
	pipe := os.NewFile(uintptr(3), "pipe")

	initArgsJsonStr, err := io.ReadAll(pipe)
	if err != nil {
		return err
	}
	args := &initArgs{}
	if err := json.Unmarshal(initArgsJsonStr, args); err != nil {
		return err
	}
	// 给当前进程设置新的net namespaec
	newNsfd, err := netns.GetFromName(args.NetnsName)
	if err != nil {
		return errors.Wrapf(err, "fail to get net fd %s", args.NetnsName)
	}
	if err := unix.Setns(int(newNsfd), syscall.CLONE_NEWNET); err != nil {
		return errors.Wrap(err, "fail to set net ")
	}
	slog.Debug("set ns", "unique id ", newNsfd.UniqueId())

	if err := setUpMount(args.MountRoot); err != nil {
		return err
	}

	command := args.Args
	path, err := exec.LookPath(command[0])
	if err != nil {
		return err
	}
	slog.Info("LookPath", "path", path)
	syscall.Sethostname([]byte(os.Getenv(args.Hostname)))
	if err := syscall.Exec(path, command[0:], os.Environ()); err != nil {
		return fmt.Errorf("syscall exec %v", err)
	}
	return nil
}

/*
*
Init 挂载点
*/
func setUpMount(mountRoot string) error {

	slog.Info("setUpMount", "Current location ", mountRoot)

	if err := pivotRoot(mountRoot); err != nil {
		return errors.WithStack(err)
	}

	//mount proc 此时的根目录已经改变了，所以挂载的是新root下的/proc 不是宿主机的
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		return errors.Wrap(err, "syscall.Mount proc")
	}

	// tmpfs 就是把raw当作硬盘，可以提升应用速度
	return errors.Wrap(syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755"), "syscall.Mount tmpfs")
}

func pivotRoot(root string) error {
	/*
			https://github.com/taikulawo/wwcdocker/issues/3
			https://man7.org/linux/man-pages/man2/pivot_root.2.html#:~:text=The%20propagation%20type,another%20mount%20namespace.
			The propagation type of the parent mount of new_root and the
		    parent mount of the current root directory must not be
		    MS_SHARED; similarly, if put_old is an existing mount point,
		    its propagation type must not be MS_SHARED.  These
		    restrictions ensure that pivot_root() never propagates any
		    changes to another mount namespace.
	*/
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return errors.Wrap(err, "fail to set root flags MS_PRIVATE")
	}
	// 重新mount一下当前root 以区分出不同的 mount namespace  旧root的mount namespace 应该是父进程的
	// mount bind 将前一个目录挂载到后一个目录上，所有对后一个目录的访问其实都是对前一个目录的访问
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return errors.Wrap(err, "mount rootfs to itself")
	}

	// 使用provt_root 改变进程的根目录 old_root -> new_root
	// 创建 rootfs/.pivot_root 存储 old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return errors.Wrap(err, "Mkdir .pivot_root")
	}

	// pivot_root 到新的rootfs, 现在老的 old_root 是挂载在rootfs/.pivot_root
	// 挂载点现在依然可以在mount命令中看到
	// PivotRoot(newroot string, putold string)
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return errors.Wrapf(err, "pivotRoot pivot_root root %s pivotDir %s", root, pivotDir)
	}

	// 修改当前的工作目录到根目录
	// pivot_root 不会修改当前工作区 因此需要用 chdir
	if err := syscall.Chdir("/"); err != nil {
		return errors.Wrap(err, "chdir /")
	}

	// 删除前先 unmount 旧root path 防止删除后出现问题
	pivotDir = filepath.Join("/", ".pivot_root")
	// umount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return errors.Wrap(err, "unmount pivot_root dir")
	}
	// 删除临时文件夹
	return errors.Wrap(os.Remove(pivotDir), "remove pivotDir")
}
