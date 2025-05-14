package balancer

import (
	"fmt"
	"sync"
	"log"

	"van/cloud-balancer/internal/servers"
)

type RRBalancer struct {
	servers []servers.Server
	current_server int
	wg sync.WaitGroup
	mu sync.Mutex
}

func (rrb *RRBalancer) ConnectToServers(cfg string) error {
	srvs, err := servers.ConnectToServers(cfg, &rrb.wg)
	if err != nil {
		return fmt.Errorf("error connection to servers: %w", err)
	}

	rrb.servers = srvs

	return nil
}

func (rrb *RRBalancer) Redirect() string {
	rrb.mu.Lock()
	defer rrb.mu.Unlock()
	
	address := ""
	if rrb.servers[rrb.current_server].IsFree() {
		address = rrb.servers[rrb.current_server].Address
	}

	rrb.current_server = (rrb.current_server + 1) % len(rrb.servers)

	return address
}

func (rrb *RRBalancer) Close() {
	// Остановка серверов
	log.Println("Closing started")
	
	for i := range rrb.servers {
		rrb.servers[i].Stop <- true
	}
	rrb.wg.Wait()

	log.Println("Closing done")
}
