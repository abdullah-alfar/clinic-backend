package db

import (
	"clinic-backend/internal/config"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

func NewDB() (*sql.DB, error) {
	cfg := config.LoadDBConfig()

	dsn := cfg.DSN()
	if dsn == "" {
		return nil, fmt.Errorf("unsupported DB_CONNECTION: %s", cfg.Connection)
	}

	db, err := sql.Open(cfg.Connection, dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
