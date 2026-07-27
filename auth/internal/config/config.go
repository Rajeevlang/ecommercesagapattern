package config

import (
	"os"
	"time"
)

type Config struct {
	GRPCPort        string
	DatabaseURL     string
	JWTSecret       string
	TokenExpiration time.Duration
	Environment     string
}

func LoadConfig() *Config {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://root:secretpassword@localhost:5432/auth_db?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-ecom-key-change-in-production"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	return &Config{
		GRPCPort:        port,
		DatabaseURL:     dbURL,
		JWTSecret:       jwtSecret,
		TokenExpiration: 24 * time.Hour,
		Environment:     env,
	}
}

