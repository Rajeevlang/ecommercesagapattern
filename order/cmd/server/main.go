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

	"github.com/Rajeevlang/ecommercesagapattern/order/config"
	ordergrpc "github.com/Rajeevlang/ecommercesagapattern/order/internals/grpc"
	orderpub "github.com/Rajeevlang/ecommercesagapattern/order/internals/publisher"
	"github.com/Rajeevlang/ecommercesagapattern/order/internals/repo"
	"github.com/Rajeevlang/ecommercesagapattern/order/internals/service"
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	catalogv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/catalog/v1"
	orderv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/order/v1"
	paymentv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Order Microservice [%s]...", cfg.Environment)

	// 1. Initialize PostgreSQL Connection Pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := repo.InitPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL database (%s): %v", cfg.DatabaseURL, err)
	}
	log.Println("Successfully connected to PostgreSQL database.")
	defer pool.Close()

	orderRepo := repo.NewRepoHandler(pool)

	// 2. Establish gRPC client connections to Catalog, Payment, and Account services
	log.Printf("Connecting to Catalog Service at %s...", cfg.CatalogServiceURL)
	catalogConn, err := grpc.NewClient(cfg.CatalogServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Catalog Service: %v", err)
	}
	defer catalogConn.Close()
	catalogClient := catalogv1.NewCatalogServiceClient(catalogConn)

	log.Printf("Connecting to Payment Service at %s...", cfg.PaymentServiceURL)
	paymentConn, err := grpc.NewClient(cfg.PaymentServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Payment Service: %v", err)
	}
	defer paymentConn.Close()
	paymentClient := paymentv1.NewPaymentServiceClient(paymentConn)

	log.Printf("Connecting to Account Service at %s...", cfg.AccountServiceURL)
	accountConn, err := grpc.NewClient(cfg.AccountServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Account Service: %v", err)
	}
	defer accountConn.Close()
	accountClient := accountv1.NewAccountServiceClient(accountConn)

	// Initialize Kafka publisher
	kafkaPub := orderpub.NewKafkaPublisher(cfg.KafkaBrokers)

	// 3. Initialize Domain Service (containing Saga Orchestration logic)
	orderSvc := service.NewOrderService(orderRepo, catalogClient, paymentClient, accountClient, kafkaPub)

	// 4. Initialize gRPC Transport Handler
	grpcHandler := ordergrpc.NewGrpcHandler(orderSvc)

	// 5. Start gRPC Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	// Channel to capture termination signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Order gRPC Server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Order gRPC Server gracefully...")
	grpcServer.GracefulStop()
	log.Println("Order gRPC Server stopped.")
}
