package db

import "time"

type MariaDbConfig struct {
	Dsn             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration // seconds
}
