package network

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/kehaha-5/go-low-level-simple-docker/common"

	"github.com/coreos/go-iptables/iptables"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

/*
创建网桥
ip link add name bridge0 type bridge
给网桥分配子网ip
ip addr add 192.168.0.1/24(subnet) dev bridge0
启动网桥
ip link set bridge0 up
*/
const (
	defalutNetworkPath string = "/root/runc/runEnv/network/"
	// -p protocol -m match --dport 选项只对 TCP 协议有效 -j DNAT 改变数据包的目的地址 --to-destination 目标地址
	iptCommand string = "-p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s"
)

var (
	networks = map[string]*Network{}      // networkName => network
	dirvers  = map[string]NetworkDriver{} // dirvername => networkDriver
)

// 每个容器中跟网络链接的信息结构体
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

// 每一个驱动中有同子网网络 192.168.0.0/24 172.17.0.0/24 等
type Network struct {
	Id      string
	Name    string
	IpRange *net.IPNet //192.168.0.0/24
	Driver  string
}

func (t *Network) dump() error {
	nsJson, err := json.Marshal(t)
	if err != nil {
		return errors.WithStack(err)
	}
	err = os.WriteFile(path.Join(defalutNetworkPath, t.Name), nsJson, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (t *Network) load() error {
	nsJson, err := os.ReadFile(path.Join(defalutNetworkPath, t.Name))
	if err != nil {
		return errors.WithStack(err)
	}

	if err = json.Unmarshal(nsJson, t); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (t *Network) remove() error {
	return errors.WithStack(os.RemoveAll(path.Join(defalutNetworkPath, t.Name)))
}

func (t *Network) wirteInfoToTabwriter(w *tabwriter.Writer) {
	// "ID\tNAME\tIpRange\tDriver\n"
	fmt.Fprintf(
		w, "%s\t%s\t%s\t%s\n",
		t.Id,
		t.Name,
		t.IpRange,
		t.Driver,
	)
}

func Init() error {
	if exist, _ := common.PathExist(defalutNetworkPath); !exist {
		if err := os.MkdirAll(defalutNetworkPath, 0644); err != nil {
			return errors.WithStack(err)
		}
	}

	filepath.Walk(defalutNetworkPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return errors.WithStack(err)
		}
		if strings.HasSuffix(path, "/") {
			return nil
		}
		if info.IsDir() {
			return filepath.SkipDir
		}

		_, file := filepath.Split(path)

		network := &Network{
			Name: file,
		}
		if err := network.load(); err != nil {
			slog.Error("network->init->load", "err", err)
		}
		networks[file] = network
		return nil
	})
	bridgeDirver := &DridgeNetworkDriver{}
	dirvers[bridgeDirver.Name()] = bridgeDirver
	return nil
}

func CreateNetwork(driver, subnet, name string) error {
	// subnet string to RFC 4632 and RFC 4291.
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return errors.WithStack(err)
	}

	driverFace, eixst := dirvers[driver]
	if !eixst {
		return errors.WithStack(fmt.Errorf("dirver name %s not eixst", driver))
	}

	aip, err := ipAllocator.Allocate(ipNet)
	if err != nil {
		return errors.Wrap(err, "failed ip alloca")
	}
	ipNet.IP = aip

	network, err := driverFace.Create(ipNet.String(), name)
	if err != nil {
		defer ipAllocator.Release(ipNet, &aip)
		return errors.WithStack(err)
	}
	return network.dump()
}

func RemoveNetwork(name string) error {
	n, eixst := networks[name]
	if !eixst {
		return errors.WithStack(fmt.Errorf("netwrok name %s not exist", name))
	}
	ipAllocator.Release(n.IpRange, &n.IpRange.IP)
	d := dirvers[n.Driver]

	if err := d.Delete(n.Name); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(n.remove())
}

func ShowAllNetworks(w *tabwriter.Writer) {
	for _, itme := range networks {
		itme.wirteInfoToTabwriter(w)
	}
}

func Connect(networkName string, containerId string, containerPid string, portMapping []string) error {
	network, exist := networks[networkName]
	if !exist {
		return fmt.Errorf("network name %s not exist", networkName)
	}

	// 为该容器分配ip
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", containerId, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: portMapping,
	}

	// 创建veth link 并且把veth1 link到对应的网络中
	if err := dirvers[network.Driver].Connect(network, ep); err != nil {
		return errors.WithStack(err)
	}

	// 配置容器网络
	if err := configContainerNetwork(ep, containerPid, network.IpRange, &ip); err != nil {
		defer ipAllocator.Release(network.IpRange, &ip)
		return errors.WithStack(err)
	}

	// 配置宿主机网络端口映射
	if err := configMapping(ep); err != nil {
		defer ipAllocator.Release(network.IpRange, &ip)
		return errors.WithStack(err)
	}
	return nil
}

func configContainerNetwork(ep *Endpoint, containerPid string, gwIpNet *net.IPNet, containerIp *net.IP) error {

	// 获取Connect配置的veth
	vethL, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return errors.Wrap(err, "fail to get veth link")
	}

	// 进入容器netse 下面的代码都是在容器的net namespace的环境下执行的
	defer enterContainerNetns(&vethL, containerPid)()

	containerIpNet := &net.IPNet{IP: gwIpNet.IP, Mask: gwIpNet.Mask}
	containerIpNet.IP = *containerIp
	ipnet, err := netlink.ParseIPNet(containerIpNet.String())
	if err != nil {
		return errors.Wrap(err, "fialed to get ParseIPNet")
	}
	addr := &netlink.Addr{IPNet: ipnet, Label: "", Flags: 0, Scope: 0, Peer: nil, Broadcast: net.IPv4(0, 0, 0, 0), PreferedLft: 0, ValidLft: 0}
	// 把ip添加到vethlink中
	if err := netlink.AddrAdd(vethL, addr); err != nil {
		return errors.Wrap(err, "fail to add ip to container veth")
	}
	// 启动veth
	if err := netlink.LinkSetUp(vethL); err != nil {
		return errors.Wrap(err, "fail to add setup veth")
	}

	// 启动lo
	loLink, err := netlink.LinkByName("lo")
	if err != nil {
		return errors.Wrap(err, "fail to get lo link")
	}
	if err := netlink.LinkSetUp(loLink); err != nil {
		return errors.Wrap(err, "fail to add setup loLink")
	}
	// 添加路由，所有流量走veth
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	defaultRoute := &netlink.Route{
		LinkIndex: vethL.Attrs().Index,
		Gw:        gwIpNet.IP, //网关要设置成宿主机网卡的ip
		Dst:       cidr,
	}

	return errors.Wrapf(netlink.RouteAdd(defaultRoute), "fail to add route %s", defaultRoute.String())
}

func enterContainerNetns(enLink *netlink.Link, containerPid string) func() {

	// 找到容器的Net Namespace
	// /proc/[pid]/ns/net 打开这个文件的文件描述符就可以来操作 Net Namespace
	// 而Containerinfo 中的 PID，即容器在宿主机上映射的进程 ID
	// 它对应的 /proc/[pid]/ns/net 就是容器内部的Net Namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", containerPid), os.O_RDONLY, 0)
	if err != nil {
		slog.Error("enterContainerNetns", fmt.Errorf("fail to open pid %s ns file", containerPid))
	}
	// 锁定当前程序所执行的线程，如果不锁定操作系统线程的话
	// Go 语言的 goroutine 可能会被调度到别的线程上去
	// 就不能保证一直在所需要的网络空间中了
	// 所以调用 runtime.LockOSThread 时要先锁定当前程序执行的线
	runtime.LockOSThread()
	nsFD := f.Fd()
	// 把veth设置进入目标容器的net namespace中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		slog.Error("fail to set link to net namespeace")
	}

	// 切换net namespace
	orgin, err := netns.Get()
	if err != nil {
		slog.Error("fail to get orgin error")
	}

	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		slog.Error("fail to set net namespace configuration")
	}

	return func() {
		netns.Set(orgin)
		orgin.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

func configMapping(ep *Endpoint) error {
	ipt, err := iptables.New()
	for _, item := range ep.PortMapping {
		portMap := strings.Split(item, ":")
		if len(portMap) != 2 {
			slog.Error("configMapping", "port mapping error format %v !", item)
			continue
		}
		// 通过iptables设置宿主机和容器网络的ip端口映射
		if err != nil {
			slog.Error("new ipt error %v", err)
			continue
		}
		if err := ipt.Append("nat", "PREROUTING", strings.Split(fmt.Sprintf(iptCommand, portMap[0], ep.IPAddress.String(), portMap[1]), " ")...); err != nil {
			slog.Error("fail to set ipt command %v", err)
			continue
		}
		rule, err := ipt.List("nat", "PREROUTING")
		if err != nil {
			slog.Debug("fail to get rule %v", err)
			continue
		}
		slog.Debug("configMapping", "ipt rule %s", rule)
		if err := ipt.ClearAll(); err != nil {
			return errors.Wrap(err, "fail to clear ipt")
		}
	}
	return nil
}
