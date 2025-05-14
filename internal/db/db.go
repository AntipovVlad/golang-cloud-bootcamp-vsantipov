package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type PostgreDBParams struct {
	DBName string
	Host string
	User string
	Password string
}

var DB *sql.DB

func verifyTableExists() (bool, error) {
	const table = "users"

	var exists bool
	q := `
        SELECT EXISTS (
            SELECT 1 
            FROM information_schema.tables 
            WHERE table_schema = 'public' 
            AND table_name = $1
        )
    `
	if err := DB.QueryRow(q, table).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func createTable() error {
	var err error

	createQuery := `
		create table users(
			id      			SERIAL PRIMARY KEY,
			name    			VARCHAR(100) UNIQUE,
			api_key 			VARCHAR(64) UNIQUE,
			capacity			INTEGER,
			current_capacity	INTEGER,
			rate_per_sec 		INTEGER
	)`
	
	_, err = DB.Exec(createQuery)

	return err
}

func InitDB(config PostgreDBParams) error {
	connStr := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable", 
		config.Host, config.DBName, config.User, config.Password)
	
	var err error
	var exists bool
	
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to open db connection: %w", err)
	}

	exists, err = verifyTableExists()
	if err != nil {
		return fmt.Errorf("failed to verify table exists: %w", err) 
	}
	if !exists {
		if err = createTable(); err != nil {
			return fmt.Errorf("failed to create table: %w", err) 
		}
	}

	return nil
}
