package network

import (
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/kehaha-5/go-low-level-simple-docker/common"

	"github.com/coreos/go-iptables/iptables"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

type DridgeNetworkDriver struct {
}

func (t *DridgeNetworkDriver) Name() string {
	return "bridge"
}

func (t *DridgeNetworkDriver) Create(subnet string, bridgeName string) (*Network, error) {
	ip, ipRange, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ipRange.IP = ip
	n := &Network{
		Name:    bridgeName,
		IpRange: ipRange,
		Driver:  t.Name(),
		Id:      common.RangeStr(12),
	}

	// add bridge
	if interfaceIsExist(bridgeName) {
		return nil, errors.WithStack(fmt.Errorf("interface name has existed"))
	}

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeName,
		},
	}

	// ip link add name ${bridgeName} type bridge
	if err := netlink.LinkAdd(bridge); err != nil {
		return nil, errors.Wrapf(err, "bridge creation failed for bridge %s", bridgeName)
	}
	// add ip to bridge
	netAddr, err := netlink.ParseAddr(ipRange.String())
	if err != nil {
		return nil, errors.Wrap(err, "error ParseAddr")
	}
	// it will be automatically computed based on the IP mask ip addr add (subnet) dev ${bridgeName}
	if err = netlink.AddrAdd(bridge, netAddr); err != nil {
		return nil, errors.Wrap(err, "set ip to bridge")
	}
	// Bring up the bridge interface
	if err := netlink.LinkSetUp(bridge); err != nil {
		return nil, errors.Wrap(err, "setting bridge up")
	}
	ipt, err := iptables.New()
	if err != nil {
		return nil, errors.Wrap(err, "fail to new ipt")
	}
	_, ipRangeForIpt, _ := net.ParseCIDR(subnet) //要生成类似 172.47.0.0/24 或 192.168.1.0/24 不能添加ip 如 192.168.1.1/24
	if err := ipt.Append("nat", "POSTROUTING", "-s", ipRangeForIpt.String(), "-j", "MASQUERADE"); err != nil {
		return nil, errors.Wrap(err, "fail to set ipt command")
	}
	rule, err := ipt.List("nat", "PREROUTING")
	if err != nil {
		slog.Debug("fail to get rule", err)
	}
	slog.Debug("configMapping", "ipt rule ", rule)
	return n, nil
}

func (t *DridgeNetworkDriver) Delete(bridgeName string) error {
	link, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(netlink.LinkDel(link))
}

func (t *DridgeNetworkDriver) Connect(n *Network, ep *Endpoint) error {
	// 获取link对象
	link, err := netlink.LinkByName(n.Name)
	if err != nil {
		return errors.WithStack(err)
	}
	// 创建veth
	vethL := netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: ep.ID[:5], //宿主机显示名称
		},
		PeerName: "cif-" + ep.ID[:5], //容器显示名称
	}
	//因为上面指定了 link 的MasterIndex 是网络对应的 Linux Bridge
	//所以Veth 的一端就己经挂载到了网络对应的 Linux Bridge 上
	vethL.MasterIndex = link.Attrs().Index
	if err := netlink.LinkAdd(&vethL); err != nil {
		return errors.WithStack(err)
	}
	ep.Device = vethL
	if err := netlink.LinkSetUp(&vethL); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (t *DridgeNetworkDriver) DisConnect() error {
	return nil
}

// 判断改名词的接口是否存在 ture 存在 false 不存在
func interfaceIsExist(name string) bool {
	_, err := net.InterfaceByName(name)
	if err == nil || strings.Contains(err.Error(), "no such network interface") {
		return false
	}
	return true
}
