package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"van/cloud-balancer/internal/balancer"
	"van/cloud-balancer/internal/db"
	"van/cloud-balancer/internal/users"
	"van/cloud-balancer/internal/web"

	"github.com/joho/godotenv"
)

func main() {
	// Инициализация БД
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
	if err := db.InitDB(db.PostgreDBParams{
		DBName:   os.Getenv("DB_NAME"),
		Host:     os.Getenv("DB_HOST"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
	}); err != nil {
		log.Fatal(err)
	}
	defer db.DB.Close()

	// Настройки сервера
	var (
		mux = http.NewServeMux()
		srv = &http.Server{
			Addr:    ":8080",
			Handler: mux,
		}
	)
	// API для работы с профилем пользователя
	mux.HandleFunc("/user/create", web.CreateUserHandler)
	mux.HandleFunc("/user/tokens/profile", web.UserAuth(web.GetProfileHandler))
	mux.HandleFunc("/user/tokens/change", web.UserAuth(web.ChangeTokensHandler))
	// Балансировщик
	b := &balancer.RRBalancer{}
	if err := b.ConnectToServers("servers.yaml"); err != nil {
		log.Fatal(err)
	}
	pcache := web.NewProxyCache()
	mux.HandleFunc("/", web.UserAuth(web.BalancerHandler(b, pcache)))

	// Пополнение токенов
	interval := 1 * time.Second
	ticker := time.NewTicker(interval)
	go func() {
		for {
			<-ticker.C
			if err := users.UpdateTokens(); err != nil {
				log.Println(fmt.Errorf("failed to update tokens: %w", err))
			}
		}
	}()

	// Обработка сигналов.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// Ожидание сигнала завершения.
		sig := <-sigChan
		log.Println("Got signal:", sig)

		// Выполнение функций при завершении.
		ticker.Stop()
		b.Close()
		// Завершение работы сервера
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Println(fmt.Errorf("error server shutdown: %w", err))
		} else {
			log.Println("Server shutdnow success")
		}
	}()

	// Запуск сервера.
	log.Println("Running server at port 8080...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println(fmt.Errorf("error server runtime: %w", err))
	}
}
