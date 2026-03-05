package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenFromEnv() (*sql.DB, error) {
	dsn := DSNFromEnv()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

func DSNFromEnv() string {
	host := envOr("PGHOST", "localhost")
	port := envOr("PGPORT", "5432")
	user := envOr("PGUSER", "marketnews")
	pass := envOr("PGPASSWORD", "marketnews")
	db := envOr("PGDATABASE", "marketnews")
	sslmode := envOr("PGSSLMODE", "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, pass, db, sslmode)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
