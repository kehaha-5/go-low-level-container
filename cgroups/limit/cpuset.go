package limit

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

type CpusetItem struct {
	cgfilepath string //保存当前资源组root路径
	isApply    bool
}

func (*CpusetItem) GetType() string {
	return "cpuset"
}

func (t *CpusetItem) CreateLimitFile(name string, conf *ResourceConfig) error {
	cgfilepath, err := findAndCreateCgroupFilePath(t.GetType(), name, true)
	if err == nil {
		if conf.Cpuset != 0 {
			if err = os.WriteFile(path.Join(cgfilepath, limitCpusetFilename), []byte(string(rune(conf.Cpuset))), 0664); err == nil {
				t.cgfilepath = cgfilepath
				t.isApply = true
			} else {
				return fmt.Errorf("create cg file error %v", err)
			}
		}
		t.isApply = false
		return nil
	}
	return err
}

func (t *CpusetItem) Apply(pid int) error {
	if !t.isApply {
		return nil
	}
	if t.cgfilepath == "" {
		return fmt.Errorf("create the limit file before use this pls")
	}
	if err := os.WriteFile(path.Join(t.cgfilepath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("set cgroup proc fail %v type is %s", err, t.GetType())
	}
	return nil
}

func (t *CpusetItem) Remove() error {
	if !t.isApply {
		return nil
	}
	return os.RemoveAll(t.cgfilepath)
}
