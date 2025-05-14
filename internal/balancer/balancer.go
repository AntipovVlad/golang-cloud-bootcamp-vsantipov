package balancer

type Balancer interface {
	ConnectToServers(cfg string) error
	Redirect() string
	Close()
}
