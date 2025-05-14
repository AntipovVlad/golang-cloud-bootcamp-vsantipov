package web

import (
	"context"
	"errors"
	"encoding/json"
	"log"
	"net/http"

	"van/cloud-balancer/internal/users"
)

func UserAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api_key := r.Header.Get("Authorization")
		
		var u users.User
		if err := u.Auth(api_key); err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), users.UserContextKey, &u)
		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}

func ChangeTokensHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

	auth_user := r.Context().Value(users.UserContextKey).(*users.User)

	var u users.TokensUser
	if err := decodeJSONBody(w, r, &u); err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	auth_user.ChangeTokens(&u)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auth_user)
}

func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

	auth_user := r.Context().Value(users.UserContextKey).(*users.User)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auth_user)
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

	var request_user users.CreateUserRequest

	if err := decodeJSONBody(w, r, &request_user); err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
		return
	}

	response_user, err := users.CreateUser(request_user.Name)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response_user)
}
