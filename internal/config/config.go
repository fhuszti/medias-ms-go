package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log"
	"time"
)

type Settings struct {
	MariaDBDSN      string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ServerPort      int
}

func Load() (*Settings, error) {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found; proceeding with OS environment variables")
	}

	viper.AutomaticEnv()

	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: could not read .env file: %v", err)
	}

	if !viper.IsSet("MARIADB_DSN") {
		return nil, fmt.Errorf("MARIADB_DSN is required")
	}
	if !viper.IsSet("MARIADB_MAX_OPEN_CONN") {
		return nil, fmt.Errorf("MARIADB_MAX_OPEN_CONN is required")
	}
	if !viper.IsSet("MARIADB_MAX_IDLE_CONNS") {
		return nil, fmt.Errorf("MARIADB_MAX_IDLE_CONNS is required")
	}
	if !viper.IsSet("MARIADB_CONN_MAX_LIFETIME") {
		return nil, fmt.Errorf("MARIADB_CONN_MAX_LIFETIME is required")
	}
	if !viper.IsSet("SERVER_PORT") {
		return nil, fmt.Errorf("SERVER_PORT is required")
	}

	return &Settings{
		MariaDBDSN:      viper.GetString("MARIADB_DSN"),
		MaxOpenConns:    viper.GetInt("MARIADB_MAX_OPEN_CONN"),
		MaxIdleConns:    viper.GetInt("MARIADB_MAX_IDLE_CONNS"),
		ConnMaxLifetime: time.Duration(viper.GetInt("MARIADB_CONN_MAX_LIFETIME")) * time.Second,
		ServerPort:      viper.GetInt("SERVER_PORT"),
	}, nil
}
