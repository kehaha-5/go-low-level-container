package cgroups

import (
	"github.com/kehaha-5/go-low-level-container/cgroups/limit"
)

type CgroupManager struct {
	// cgroup在hierarchy中的路径 相当于创建的cgroup目录相对于root cgroup目录的路径
	Path string
	// 资源配置
	Resource *limit.ResourceConfig
	// 每个资源的item
	resourceItem []limit.ResourceItem
}

func NewCgroupManager(path string) *CgroupManager {
	ins := &CgroupManager{Path: path}
	ins.resourceItem = []limit.ResourceItem{
		&limit.CpuItem{},
		&limit.CpusetItem{},
		&limit.MemoryItem{},
	}
	return ins
}

// 设置cgroup资源限制
func (t *CgroupManager) Set(res *limit.ResourceConfig) error {
	for _, subSysIns := range t.resourceItem {
		if err := subSysIns.CreateLimitFile(t.Path, res); err != nil {
			return err
		}
	}
	return nil
}

// 将进程pid加入到这个cgroup中
func (t *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range t.resourceItem {
		if err := subSysIns.Apply(pid); err != nil {
			return err
		}
	}
	return nil
}

// 释放cgroup
func (t *CgroupManager) Destroy() error {
	for _, subSysIns := range t.resourceItem {
		if err := subSysIns.Remove(); err != nil {
			return err
		}
	}
	return nil
}
