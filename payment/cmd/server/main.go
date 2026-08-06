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

	"github.com/Rajeevlang/ecommercesagapattern/payment/config"
	paymentgrpc "github.com/Rajeevlang/ecommercesagapattern/payment/internal/grpc"
	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/ports"
	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/repository/postgres"
	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/service"
	paymentv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Payment Microservice [%s]...", cfg.Environment)

	var repo ports.PaymentRepository
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Initialize Database Repository (or fallback to in-memory)
	pool, err := postgres.InitPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL database (%v). Falling back to in-memory repository.", err)
		repo = memory.NewInMemoryPaymentRepository()
	} else {
		log.Println("Successfully connected to PostgreSQL database.")
		defer pool.Close()
		repo = postgres.NewPaymentRepository(pool)
	}

	// 2. Initialize Domain Service
	paymentSvc := service.NewPaymentService(repo)

	// 3. Initialize gRPC Handler
	grpcHandler := paymentgrpc.NewGrpcHandler(paymentSvc)

	// 4. Start gRPC Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	// Signal capture for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Payment gRPC Server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Payment gRPC Server gracefully...")
	grpcServer.GracefulStop()
	log.Println("Payment Microservice stopped.")
}
