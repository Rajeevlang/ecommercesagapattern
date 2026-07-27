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

	"github.com/Rajeevlang/ecommercesagapattern/account/config"
	accountgrpc "github.com/Rajeevlang/ecommercesagapattern/account/internal/grpc"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/ports"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/repository/postgres"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/service"
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Account Microservice [%s] on port %s...", cfg.Environment, cfg.GRPCPort)

	var repo ports.AccountRepository
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.InitPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL database (%v). Falling back to in-memory repository.", err)
		repo = memory.NewInMemoryAccountRepository()
	} else {
		log.Println("Successfully connected to PostgreSQL database.")
		defer pool.Close()
		repo = postgres.NewPostgresAccountRepository(pool)
	}

	accountSvc := service.NewAccountService(repo)
	grpcHandler := accountgrpc.NewAccountGRPCHandler(accountSvc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	accountv1.RegisterAccountServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Account gRPC Server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Account gRPC Server gracefully...")
	grpcServer.GracefulStop()
	log.Println("Account gRPC Server stopped.")
}
