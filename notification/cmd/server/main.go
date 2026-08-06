package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/notification/config"
	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/event"
	notificationgrpc "github.com/Rajeevlang/ecommercesagapattern/notification/internals/grpc"
	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/ports"
	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/providers/email"
	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/providers/sms"
	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/repository/postgres"
	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/service"
	notificationv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Notification Microservice [%s]...", cfg.Environment)

	var repo ports.NotificationRepository
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Initialize repository (postgres pool or fallback to in-memory)
	pool, err := postgres.InitPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL database (%v). Falling back to in-memory repository.", err)
		repo = memory.NewInMemoryNotificationRepository()
	} else {
		log.Println("Successfully connected to PostgreSQL database.")
		defer pool.Close()
		repo = postgres.NewNotificationRepository(pool)
	}

	// 2. Initialize email & SMS providers
	var emailProvider ports.EmailProvider
	if cfg.EmailProvider == "smtp" {
		log.Printf("Using SMTP Email Provider: %s:%s\n", cfg.SMTPHost, cfg.SMTPPort)
		emailProvider = email.NewSMTPEmailProvider(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom, cfg.SMTPUser, cfg.SMTPPass)
	} else {
		log.Println("Using Mock Email Provider")
		emailProvider = email.NewMockEmailProvider()
	}
	smsProvider := sms.NewMockSMSProvider()

	// 3. Initialize Domain Service & background workers
	notificationSvc := service.NewNotificationService(repo, emailProvider, smsProvider)
	notificationSvc.StartRetryWorker(10 * time.Second)
	defer notificationSvc.StopRetryWorker()

	// 4. Initialize Kafka Consumer
	kafkaConsumer := event.NewKafkaConsumer(cfg.KafkaBrokers, cfg.TemplatesDir, notificationSvc)
	go func() {
		// Attempt connection, if kafka is offline, print warning but keep running
		if err := kafkaConsumer.Start(context.Background()); err != nil {
			log.Printf("Warning: Kafka consumer failed to start (is Kafka offline?): %v\n", err)
		}
	}()
	defer kafkaConsumer.Stop()

	// 5. Initialize gRPC Transport Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	grpcHandler := notificationgrpc.NewGrpcHandler(notificationSvc)
	grpcServer := grpc.NewServer()
	notificationv1.RegisterNotificationServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	// Channel to capture termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Notification gRPC Server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Notification gRPC Server gracefully...")
	grpcServer.GracefulStop()
	log.Println("Notification Microservice stopped.")
}
