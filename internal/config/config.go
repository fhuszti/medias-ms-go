package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Settings struct {
	MariaDBDSN      string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ServerPort      int
	MinioAccessKey  string
	MinioSecretKey  string
	MinioEndpoint   string
	MinioUseSSL     bool
	Buckets         []string
	ImagesSizes     []int
	RedisAddr       string
	RedisPassword   string
	JWTPublicKey    string
}

func Load() (*Settings, error) {
	log.Println("loading env variables...")

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
	if !viper.IsSet("MINIO_ACCESS_KEY") {
		return nil, fmt.Errorf("MINIO_ACCESS_KEY is required")
	}
	if !viper.IsSet("MINIO_SECRET_KEY") {
		return nil, fmt.Errorf("MINIO_SECRET_KEY is required")
	}
	if !viper.IsSet("MINIO_ENDPOINT") {
		return nil, fmt.Errorf("MINIO_ENDPOINT is required")
	}
	if !viper.IsSet("MINIO_USE_SSL") {
		return nil, fmt.Errorf("MINIO_USE_SSL is required")
	}
	if !viper.IsSet("BUCKETS") {
		return nil, fmt.Errorf("BUCKETS is required")
	}
	if !viper.IsSet("IMAGES_SIZES") {
		return nil, fmt.Errorf("IMAGES_SIZES is required")
	}

	jwtPem, err := getJWTPem()
	if err != nil {
		return nil, fmt.Errorf("could not read file from JWT_PUBLIC_KEY_PATH: %w", err)
	}

	return &Settings{
		MariaDBDSN:      viper.GetString("MARIADB_DSN"),
		MaxOpenConns:    viper.GetInt("MARIADB_MAX_OPEN_CONN"),
		MaxIdleConns:    viper.GetInt("MARIADB_MAX_IDLE_CONNS"),
		ConnMaxLifetime: time.Duration(viper.GetInt("MARIADB_CONN_MAX_LIFETIME")) * time.Second,
		ServerPort:      viper.GetInt("SERVER_PORT"),
		MinioAccessKey:  viper.GetString("MINIO_ACCESS_KEY"),
		MinioSecretKey:  viper.GetString("MINIO_SECRET_KEY"),
		MinioEndpoint:   viper.GetString("MINIO_ENDPOINT"),
		MinioUseSSL:     viper.GetBool("MINIO_USE_SSL"),
		Buckets:         getBuckets(),
		ImagesSizes:     getImagesSizes(),
		RedisAddr:       viper.GetString("REDIS_ADDR"),
		RedisPassword:   viper.GetString("REDIS_PASSWORD"),
		JWTPublicKey:    jwtPem,
	}, nil
}

func getBuckets() []string {
	bucketsSet := make(map[string]struct{})
	result := make([]string, 0)

	for _, bucket := range strings.Split(viper.GetString("BUCKETS"), ",") {
		bucket = strings.TrimSpace(bucket)
		if bucket == "" {
			continue
		}
		// Prevent duplicates
		if _, exists := bucketsSet[bucket]; !exists {
			bucketsSet[bucket] = struct{}{}
			result = append(result, bucket)
		}
	}

	// Ensure "staging" is included
	if _, exists := bucketsSet["staging"]; !exists {
		result = append(result, "staging")
	}

	return result
}

func getImagesSizes() []int {
	sizes := make([]int, 0)
	for _, size := range strings.Split(viper.GetString("IMAGES_SIZES"), ",") {
		size = strings.TrimSpace(size)
		if size == "" {
			continue
		}
		sizeInt, err := strconv.Atoi(size)
		if err != nil {
			log.Printf("Warning: could not parse image size %q: %v", size, err)
			continue
		}
		sizes = append(sizes, sizeInt)
	}
	return sizes
}

func getJWTPem() (string, error) {
	jwtKeyPath := viper.GetString("JWT_PUBLIC_KEY_PATH")
	if jwtKeyPath == "" {
		return "", nil
	}

	data, err := os.ReadFile(jwtKeyPath)
	if err != nil {
		return "", err
	}
	jwtPem := string(data)

	return jwtPem, nil
}
