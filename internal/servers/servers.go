package servers

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
	"net/http"
	"log"

	"gopkg.in/yaml.v3"
)

type ServerConf struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Config struct {
	Servers []ServerConf `yaml:"servers"`
}

type Server struct {
	mu sync.Mutex

	scfg ServerConf
	Address string
	Stop chan bool
	free bool
	TimeoutSeconds int
}

func (s *Server) IsFree() bool {
	// Безопасная проверка того, свободен ли сервер
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("Check server %s: %t", s.Address, s.free)
	return s.free
}

func (s *Server) startServer(wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Connected to server: %s", s.Address)

		s.mu.Lock()
		s.free = false
		s.mu.Unlock()

		time.Sleep(time.Duration(s.TimeoutSeconds) * time.Second)

		s.mu.Lock()
		s.free = true
		s.mu.Unlock()
		
		w.Write([]byte(fmt.Sprintf("Server %s answer", s.Address)))
    })

    srv := &http.Server{
        Addr:    fmt.Sprintf(":%d", s.scfg.Port),
        Handler: mux,
    }

    go func() {
		// Остановка сервера
        <-s.Stop
        ctx, cancel := context.WithTimeout(context.Background(), 6 * time.Second)
        defer cancel()
        if err := srv.Shutdown(ctx); err != nil {
			log.Println("error server shutdown: %w", err)
        }
    }()
	
	log.Printf("-> Name: %s, Host: %s, Port: %d\n", s.scfg.Name, s.scfg.Host, s.scfg.Port)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println("error server runtime: %w", err)
    }
	log.Printf("Stopped server %s", s.scfg.Name)
}


func ConnectToServers(cfg string, wg *sync.WaitGroup) ([]Server, error) {
	// Симуляция подключения к серверам.
	// В рамках данной тестовой задачи эта функция сама поднимает их на указанных в конфиге портах при помощи горутин.
	// В реальных условиях здесь бы была обычная установка соединения.

	data, err := os.ReadFile(cfg)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var scfg Config
	if err := yaml.Unmarshal(data, &scfg); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	var Servers []Server
	for _, sd := range scfg.Servers {
		Servers = append(Servers, Server{scfg: sd, 
										 Address: fmt.Sprintf("http://localhost:%d/", sd.Port), 
										 Stop: make(chan bool), 
										 free: true, 
										 TimeoutSeconds: 5})
	}

	log.Println("Connected to:")
	for i := range Servers {
		go Servers[i].startServer(wg)
	}

	return Servers, nil
}
