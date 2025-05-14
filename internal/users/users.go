package users

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"van/cloud-balancer/internal/db"

	"github.com/google/uuid"
)

type contextKey string

const UserContextKey contextKey = "user"
const UserContextKeyB contextKey = "userb"

// Вспомогательные структуры для аутентификации и авторизации

type CreateUserRequest struct {
	Name string `json:"name"`
}

type CreateUserResponse struct {
	Name       string `json:"name"`
	Capacity   int    `json:"capacity"`
	RatePerSec int    `json:"rate_per_sec"`
	ApiKey     string `json:"api_key"`
}

func CreateUser(name string) (*CreateUserResponse, error) {
	// Проверить существование пользователя
	unique_q := `
		SELECT 1 FROM users WHERE name=$1
	`
	if err := db.DB.QueryRow(unique_q, name).Scan(); err == nil {
		return nil, fmt.Errorf("error user already exists")
	}
	// Создание нового пользователя
	api_key := uuid.New()
	hashed_api_key := sha512.Sum512_256([]byte(api_key.String()))

	new_user_q := `
		INSERT INTO users (name, api_key, capacity, current_capacity, rate_per_sec) VALUES ($1, $2, 5, 5, 1)
	`
	_, err := db.DB.Exec(new_user_q, name, hex.EncodeToString(hashed_api_key[:]))
	if err != nil {
		return nil, err
	}

	return &CreateUserResponse{Name: name, Capacity: 5, RatePerSec: 1, ApiKey: api_key.String()}, nil
}

// Структура для работы с токенами

type TokensUser struct {
	Capacity   int `json:"capacity"`
	RatePerSec int `json:"rate_per_sec"`
}

func UpdateTokens() error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	q_update := `
        UPDATE users
        SET current_capacity = LEAST(current_capacity + rate_per_sec, capacity)
    `
	_, err = tx.Exec(q_update)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error updating tokens: %w", err)
	}

	return tx.Commit()
}

// Основная структура Пользователя, используемая для работы

type User struct {
	Name            string `json:"name"`
	Capacity        int    `json:"capacity"`
	RatePerSec      int    `json:"rate_per_sec"`
	CurrentCapacity int    `json:"current_capacity"`
}

func (u *User) Auth(api_key string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	hashed_api_key := sha512.Sum512_256([]byte(api_key))

	q := `
		SELECT name, capacity, current_capacity, rate_per_sec FROM users WHERE api_key=$1
	`
	err = tx.QueryRow(q, hex.EncodeToString(hashed_api_key[:])).Scan(&u.Name, &u.Capacity, &u.CurrentCapacity, &u.RatePerSec)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error creating new user: %w", err)
	}

	return tx.Commit()
}

func (u *User) ChangeTokens(ctu *TokensUser) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	if ctu.Capacity <= 0 && ctu.RatePerSec <= 0 {
		return fmt.Errorf("error inappropriate tokens data")
	}

	u.Capacity = ctu.Capacity
	u.RatePerSec = ctu.RatePerSec

	q_update := `
		UPDATE users SET capacity = $1, rate_per_sec = $2 WHERE name = $3
	`
	_, err = tx.Exec(q_update, u.Capacity, u.RatePerSec, u.Name)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error changing user tokens: %w", err)
	}

	return tx.Commit()
}

func (u *User) CheckCanRequest() error {
	if u.CurrentCapacity <= 0 {
		return fmt.Errorf("error token limit exceed")
	}

	return nil
}

func (u *User) RequestDone() error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	q_update := `
		UPDATE users
		SET current_capacity = GREATEST(current_capacity - 1, 0) WHERE name = $1
	`
	_, err = tx.Exec(q_update, u.Name)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error user token usage: %w", err)
	}

	return tx.Commit()
}
