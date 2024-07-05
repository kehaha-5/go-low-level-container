package container

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/kehaha-5/go-low-level-container/common"
	"github.com/pkg/errors"
)

var (
	saveImagePaths = common.ROOTPATH + "/images/"
)

func LoadImage(image string) error {
	if !common.FileExist(image) {
		return fmt.Errorf("the image %s not existed", image)
	}
	isExist, _ := common.PathExist(saveImagePaths)
	if !isExist {
		if err := os.MkdirAll(saveImagePaths, 0644); err != nil {
			return errors.Wrap(err, "fail to mkdir file")
		}
	}
	if err := exec.Command("cp", image, saveImagePaths).Run(); err != nil {
		return errors.Wrap(err, "fail to cp image file to save path")
	}

	fInfo, err := os.Stat(image)
	if err != nil {
		return errors.Wrap(err, "fail to get file info")
	}
	tz, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return errors.Wrap(err, "fail to get tz of Asia/Shanghai")
	}
	createTime := time.Now().In(tz).Format(time.RFC3339)

	imageInfo := imageInfoItem{Name: fInfo.Name(), Size: sizeHumanReadable(fInfo.Size()), ID: common.RangeStr(8), CreateTime: createTime}
	return addImage(&imageInfo)
}

func sizeHumanReadable(size int64) string {
	unit := []string{"B", "KB", "MB", "GB", "TB"}
	unitI := 0
	for 1024 <= size {
		unitI++
		size = size / 1024
	}
	return strconv.FormatInt(size, 10) + unit[unitI]
}
