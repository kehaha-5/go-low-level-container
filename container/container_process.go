package container

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// 初始化容器进程
func initContainer(tty bool) (*exec.Cmd, *os.File, error) {
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

	cmd.ExtraFiles = []*os.File{readPipe}

	return cmd, writePipe, nil
}

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
	command := strings.Split(string(args), " ")
	path, err := exec.LookPath(command[0])
	if err != nil {
		return err
	}
	slog.Info("LookPath", "path", path)

	if err := syscall.Exec(path, command[0:], os.Environ()); err != nil {
		slog.Error("syscall exec", err)
	}
	return nil
}

func newPipe() (r *os.File, w *os.File, err error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
