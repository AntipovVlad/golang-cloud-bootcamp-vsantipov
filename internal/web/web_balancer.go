package web

import (
	"log"
	"net/http"

	"van/cloud-balancer/internal/balancer"
	"van/cloud-balancer/internal/users"
)

func BalancerHandler(b balancer.Balancer, cache *ProxyCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        auth_user := r.Context().Value(users.UserContextKey).(*users.User)
        if err := auth_user.CheckCanRequest(); err != nil {
            log.Println(err)
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

		target := b.Redirect()
		if target == "" {
            log.Println("No available servers")
            http.Error(w, "No available servers", http.StatusServiceUnavailable)
            return
        }

        log.Printf("Redirect to: %s", target)

		proxy, err := cache.Get(target)
        if err != nil {
            log.Println("Invalid target URL")
            http.Error(w, "Invalid target URL", http.StatusInternalServerError)
            return
        }
        
        proxy.ServeHTTP(w, r)

        if err := auth_user.RequestDone(); err != nil {
            log.Println(err)
            http.Error(w, "Token process error", http.StatusBadRequest)
            return
        }
	}
}
