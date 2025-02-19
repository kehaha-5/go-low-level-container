package container

import (
	"os"
	"path"

	"github.com/kehaha-5/go-low-level-container/common"
)

const (
	defautlLogSavename string = "container.log"
)

var (
	defaultLogSavefilepath string = common.ROOTPATH + "/container/"
)

func createlogfilePointer(containerName string) (*os.File, error) {
	logfilepath := path.Join(defaultLogSavefilepath, containerName)
	if err := os.MkdirAll(logfilepath, 0777); err != nil {
		return nil, err
	}
	logfile := path.Join(logfilepath, defautlLogSavename)
	file, err := os.Create(logfile)
	if err != nil {
		return nil, err
	}
	return file, err
}

func GetLogByContainerName(containerName string) (string, error) {
	logfile := path.Join(defaultLogSavefilepath, containerName, defautlLogSavename)
	log, err := os.ReadFile(logfile)
	if err != nil {
		return "", err
	}
	return string(log), nil
}

func delLogByContainerName(containerName string) error {
	logfile := path.Join(defaultLogSavefilepath, containerName, defautlLogSavename)
	return os.Remove(logfile)
}
