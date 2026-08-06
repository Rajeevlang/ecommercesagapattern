package config

import "os"

type Config struct {
	GRPCPort    string
	DatabaseURL string
	Environment string
}

func LoadConfig() *Config {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50055"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://root:secretpassword@localhost:5432/payment_db?sslmode=disable"
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	return &Config{
		GRPCPort:    port,
		DatabaseURL: dbURL,
		Environment: env,
	}
}
