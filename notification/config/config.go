package config

import (
	"os"
	"strings"
)

type Config struct {
	GRPCPort      string
	DatabaseURL   string
	KafkaBrokers  []string
	TemplatesDir  string
	EmailProvider string // "mock" or "smtp"
	SMTPHost      string
	SMTPPort      string
	SMTPFrom      string
	SMTPUser      string
	SMTPPass      string
	Environment   string
}

func LoadConfig() *Config {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50056"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://root:secretpassword@localhost:5432/notification_db?sslmode=disable"
	}

	brokersStr := os.Getenv("KAFKA_BROKERS")
	var brokers []string
	if brokersStr == "" {
		brokers = []string{"localhost:9092"}
	} else {
		brokers = strings.Split(brokersStr, ",")
	}

	templatesDir := os.Getenv("TEMPLATES_DIR")
	if templatesDir == "" {
		templatesDir = "./templates"
	}

	emailProvider := os.Getenv("EMAIL_PROVIDER")
	if emailProvider == "" {
		emailProvider = "mock"
	}

	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "localhost"
	}

	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "1025" // Mailhog SMTP port
	}

	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = "noreply@sagashop.com"
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	return &Config{
		GRPCPort:      port,
		DatabaseURL:   dbURL,
		KafkaBrokers:  brokers,
		TemplatesDir:  templatesDir,
		EmailProvider: emailProvider,
		SMTPHost:      smtpHost,
		SMTPPort:      smtpPort,
		SMTPFrom:      smtpFrom,
		SMTPUser:      os.Getenv("SMTP_USER"),
		SMTPPass:      os.Getenv("SMTP_PASSWORD"),
		Environment:   env,
	}
}
