package config

import (
	"os"
	"strings"
)

type Config struct {
	GRPCPort          string
	DatabaseURL       string
	CatalogServiceURL string
	PaymentServiceURL string
	AccountServiceURL string
	KafkaBrokers      []string
	Environment       string
}

func LoadConfig() *Config {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50053"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://root:secretpassword@localhost:5432/order_db?sslmode=disable"
	}

	catalogURL := os.Getenv("CATALOG_SERVICE_URL")
	if catalogURL == "" {
		catalogURL = "localhost:50054"
	}

	paymentURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentURL == "" {
		paymentURL = "localhost:50055"
	}

	accountURL := os.Getenv("ACCOUNT_SERVICE_URL")
	if accountURL == "" {
		accountURL = "localhost:50052"
	}

	brokersStr := os.Getenv("KAFKA_BROKERS")
	var brokers []string
	if brokersStr == "" {
		brokers = []string{"localhost:9092"}
	} else {
		brokers = strings.Split(brokersStr, ",")
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	return &Config{
		GRPCPort:          port,
		DatabaseURL:       dbURL,
		CatalogServiceURL: catalogURL,
		PaymentServiceURL: paymentURL,
		AccountServiceURL: accountURL,
		KafkaBrokers:      brokers,
		Environment:       env,
	}
}
