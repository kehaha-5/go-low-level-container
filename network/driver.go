package network

// 驱动 即docker里面的bridge host 等
type NetworkDriver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(bridgeName string) error
	Connect(n *Network, ep *Endpoint) error
	DisConnect() error
}
