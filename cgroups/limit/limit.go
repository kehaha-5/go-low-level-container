package limit

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

const mountinfofile string = "/proc/self/mountinfo"
const limitCpuFilename = "cpu.shares"
const limitCpusetFilename = "cpuset.cpus"
const limitMemoryFilename = "memory.limit_in_bytes"

type ResourceConfig struct {
	Cpu    int
	Cpuset int
	Memory string
}

type ResourceItem interface {
	GetType() string                                               //获取该资源的类型
	CreateLimitFile(cgroupName string, conf *ResourceConfig) error //在资源组中创建该资源的限制文件
	Apply(pid int) error                                           //添加pid到该资源组
	Remove() error                                                 //删除真个资源组
}

// 在资源限制的文件地址中创建一个名为cgroupName的资源文件夹（可以理解为资源组）
func findAndCreateCgroupFilePath(limitType string, cgroupName string, autoCreate bool) (string, error) {
	cgrouproot, err := findCgroupRootByResType(limitType)
	if err != nil {
		return "", err
	}

	if _, err = os.Stat(path.Join(cgrouproot, cgroupName)); err == nil || (autoCreate && os.IsNotExist(err)) {
		if autoCreate {
			if err := os.MkdirAll(path.Join(cgrouproot, cgroupName), 0755); err != nil {
				return "", fmt.Errorf("create cgroup %v", err)
			}
		}
		return path.Join(cgrouproot, cgroupName), nil
	}
	return "", fmt.Errorf("cgrouproot not exist %s", path.Join(cgrouproot, cgroupName))
}

// 查找系统上设置资源限制的文件地址
func findCgroupRootByResType(limitType string) (string, error) {
	f, err := os.Open(mountinfofile)
	if err != nil {
		return "", fmt.Errorf("open mountinfofile %v", err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.Split(scanner.Text(), " ")
		for _, item := range text {
			for _, itemType := range strings.Split(item, ",") {
				if itemType == limitType {
					return text[4], nil // 第4个位置即该类型的cgroupfile文件位置
				}
			}
		}
	}
	return "", fmt.Errorf("can not find the rootfile of the %s type", limitType)
}

