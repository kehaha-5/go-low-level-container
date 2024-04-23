package container

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// 执行容器内应用进程
// 挂载/proc
func runContainerProgram() error {
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		return err
	}

	pipe := os.NewFile(uintptr(3), "pipe")

	args, err := io.ReadAll(pipe)
	if err != nil {
		return err
	}
	mountRoot := os.Getenv("mountRoot")
	if mountRoot == "" {
		return fmt.Errorf("mountRoot is empty")
	}

	if err := setUpMount(mountRoot); err != nil {
		return err
	}

	command := strings.Split(string(args), " ")
	path, err := exec.LookPath(command[0])
	if err != nil {
		return err
	}
	slog.Info("LookPath", "path", path)

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

	slog.Info("setUpMount", "Current location is", mountRoot)

	if err := pivotRoot(mountRoot); err != nil {
		return err
	}

	//mount proc 此时的根目录已经改变了，所以挂载的是新root下的/proc 不是宿主机的
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	// tmpfs 就是把raw当作硬盘，可以提升应用速度
	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")

	return nil
}

func pivotRoot(root string) error {
	// 重新mount一下当前root 以区分出不同的 mount namespace  旧root的mount namespace 应该是父进程的
	// mount bind 将前一个目录挂载到后一个目录上，所有对后一个目录的访问其实都是对前一个目录的访问
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount rootfs to itself error: %v", err)
	}

	// 使用provt_root 改变进程的根目录 old_root -> new_root
	// 创建 rootfs/.pivot_root 存储 old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}

	// pivot_root 到新的rootfs, 现在老的 old_root 是挂载在rootfs/.pivot_root
	// 挂载点现在依然可以在mount命令中看到
	// PivotRoot(newroot string, putold string)
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}

	// 修改当前的工作目录到根目录
	// pivot_root 不会修改当前工作区 因此需要用 chdir
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %v", err)
	}

	// 删除前先 unmount 旧root path 防止删除后出现问题
	pivotDir = filepath.Join("/", ".pivot_root")
	// umount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %v", err)
	}
	// 删除临时文件夹
	return os.Remove(pivotDir)
}
