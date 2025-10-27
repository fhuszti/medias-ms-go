package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

type Settings struct {
	MariaDBDSN     string
	ServerPort     int
	MinioAccessKey string
	MinioSecretKey string
	MinioEndpoint  string
	MinioUseSSL    bool
	Buckets        []string
	ImagesSizes    []int
	RedisAddr      string
	RedisPassword  string
	JWTPublicKey   string
}

func Load() (*Settings, error) {
	ctx := context.Background()

	logger.Info(ctx, "loading env variables...")

	if err := godotenv.Load(".env"); err != nil {
		logger.Info(ctx, "No .env file found; proceeding with OS environment variables")
	}

	viper.AutomaticEnv()

	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		logger.Warnf(ctx, "Warning: could not read .env file: %v", err)
	}

	mariaDBDSN, err := getMariaDBDSN()
	if err != nil {
		return nil, err
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
		MariaDBDSN:     mariaDBDSN,
		ServerPort:     viper.GetInt("SERVER_PORT"),
		MinioAccessKey: viper.GetString("MINIO_ACCESS_KEY"),
		MinioSecretKey: viper.GetString("MINIO_SECRET_KEY"),
		MinioEndpoint:  viper.GetString("MINIO_ENDPOINT"),
		MinioUseSSL:    viper.GetBool("MINIO_USE_SSL"),
		Buckets:        getBuckets(),
		ImagesSizes:    getImagesSizes(),
		RedisAddr:      viper.GetString("REDIS_ADDR"),
		RedisPassword:  viper.GetString("REDIS_PASSWORD"),
		JWTPublicKey:   jwtPem,
	}, nil
}

func getMariaDBDSN() (string, error) {
	requiredKeys := []string{
		"MARIADB_USER",
		"MARIADB_PASS",
		"MARIADB_HOST",
		"MARIADB_INTERNAL_PORT",
		"MARIADB_NAME",
	}

	for _, key := range requiredKeys {
		if !viper.IsSet(key) {
			return "", fmt.Errorf("%s is required", key)
		}
	}

	user := viper.GetString("MARIADB_USER")
	password := viper.GetString("MARIADB_PASS")
	host := viper.GetString("MARIADB_HOST")
	port := viper.GetString("MARIADB_INTERNAL_PORT")
	database := viper.GetString("MARIADB_NAME")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local", user, password, host, port, database), nil
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
	ctx := context.Background()
	sizes := make([]int, 0)
	for _, size := range strings.Split(viper.GetString("IMAGES_SIZES"), ",") {
		size = strings.TrimSpace(size)
		if size == "" {
			continue
		}
		sizeInt, err := strconv.Atoi(size)
		if err != nil {
			logger.Warnf(ctx, "Warning: could not parse image size %q: %v", size, err)
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
