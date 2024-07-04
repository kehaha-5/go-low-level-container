package container

import (
	"log/slog"
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

// 初始化容器进程
func initContainerParentWithNewWorkSpace(tty bool, volumeArg []string, containerName string, imageName string, envList []string) (*exec.Cmd, *os.File, *workSpace, error) {
	readPipe, writePipe, cmd, err := initContainerParent()
	if err != nil {
		return nil, nil, nil, err
	}
	if tty {
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
	} else {
		lopfile, err := createlogfilePointer(containerName)
		if err != nil {
			return nil, nil, nil, errors.WithStack(err)
		}
		cmd.Stdout = lopfile
		cmd.Stderr = lopfile
	}

	workSpaceInfo, err := NewWorkSpace(imageName, containerName, volumeArg)
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}
	setProcessEnv(cmd, readPipe, envList)

	return cmd, writePipe, workSpaceInfo, nil
}

func newPipe() (r *os.File, w *os.File, err error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

func initContainerParent() (*os.File, *os.File, *exec.Cmd, error) {
	readPipe, writePipe, err := newPipe()
	if err != nil {
		slog.Error("new pipe", err)
		return nil, nil, nil, err
	}

	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	return readPipe, writePipe, cmd, nil
}

func setProcessEnv(cmd *exec.Cmd, r *os.File, envList []string) {
	cmd.ExtraFiles = []*os.File{r}
	cmd.Env = append(cmd.Env, envList...)
}
