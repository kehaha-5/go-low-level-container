package network

import (
	"encoding/json"
	"go-low-level-simple-runc/common"
	"net"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

const ipamDefaultAllocatorPath = "ipam"
const ipamSaveIpAllocatorFile = "subnet.json"

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: path.Join(defalutNetworkPath, ipamDefaultAllocatorPath, ipamSaveIpAllocatorFile),
}

func (ipam *IPAM) load() error {
	if !common.FileExist(ipam.SubnetAllocatorPath) {
		filepath, _ := path.Split(ipam.SubnetAllocatorPath)
		os.MkdirAll(filepath, 0644)
		return nil
	}
	subnetJson, err := os.ReadFile(ipam.SubnetAllocatorPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(subnetJson, ipam.Subnets)
}

func (ipam *IPAM) dump() error {
	subnetJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	return os.WriteFile(ipam.SubnetAllocatorPath, subnetJson, 0644)
}

func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	// 存放网段中地址分配信息的数组
	ipam.Subnets = &map[string]string{}

	// 从文件中加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		return nil, errors.Wrap(err, "Error dump allocation info")
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	// 计算出可用地址 192.168.0.0/24 one = 24 size =32 即可计算出可以分配的地址 uint8(32-24)
	one, size := subnet.Mask.Size()

	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}

	for c := range (*ipam.Subnets)[subnet.String()] {
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			ip = subnet.IP
			for t := uint(4); t > 0; t -= 1 {
				/*
					假设 subnet.IP 是 192.168.0.0，偏移量 c 是 10。这个 c 用二进制表示是 0000 0000 0000 0000 0000 0000 0000 1010。
					对于 t = 4（最高位），(t - 1) * 8 = 24，c >> 24 = 0，ip[0] += 0。0000 0000 头8位
					对于 t = 3，(t - 1) * 8 = 16，c >> 16 = 0，ip[1] += 0000 0000 8到-16位
					对于 t = 2，(t - 1) * 8 = 8，c >> 8 = 0，ip[2] += 0。0000 0000 16-24位
					对于 t = 1，(t - 1) * 8 = 0，c >> 0 = 10，ip[3] += 10。0000 1010 24-32位
					最终，基准IP地址 192.168.0.0 变成了 192.168.0.10。
				*/
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1
			break
		}
	}

	if err = ipam.dump(); err != nil {
		return nil, errors.WithStack(err)
	}

	return
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	err := ipam.load()
	if err != nil {
		return errors.Wrap(err, "Error dump allocation info")
	}

	/*
		subnet.IP 是 192.168.0.0，表示为 [192, 168, 0, 0]。
		releaseIP 是 192.168.0.10（减去 1 之前），表示为 [192, 168, 0, 10]。
		逐步计算

		当 t = 4 时：
		releaseIP[3] 是 9，subnet.IP[3] 是 0。
		releaseIP[3] - subnet.IP[3] 是 9 - 0 = 9。
		9 << ((4 - 4) * 8) 是 9 << 0，结果是 9。
		c = 0 + 9 = 9。

		当 t = 3 时：
		releaseIP[2] 是 0，subnet.IP[2] 是 0。
		releaseIP[2] - subnet.IP[2] 是 0 - 0 = 0。
		0 << ((4 - 3) * 8) 是 0 << 8，结果是 0。
		c = 9 + 0 = 9。

		当 t = 2 时：
		releaseIP[1] 是 168，subnet.IP[1] 是 168。
		releaseIP[1] - subnet.IP[1] 是 168 - 168 = 0。
		0 << ((4 - 2) * 8) 是 0 << 16，结果是 0。
		c = 9 + 0 = 9。

		当 t = 1 时：
		releaseIP[0] 是 192，subnet.IP[0] 是 192。
		releaseIP[0] - subnet.IP[0] 是 192 - 192 = 0。
		0 << ((4 - 1) * 8) 是 0 << 24，结果是 0。
		c = 9 + 0 = 9。

		通过上述 192.168.0.0 和 192.168.0.10 的比较 得出c=9即 192.168.0.10在位图上面的位置
	*/
	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	return errors.WithStack(ipam.dump())
}
