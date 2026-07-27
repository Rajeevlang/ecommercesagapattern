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

	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/config"
	authgrpc "github.com/Rajeevlang/ecommercesagapattern/auth/internal/grpc"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/ports"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/repository/postgres"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/security"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/service"
	authv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Auth Microservice [%s]...", cfg.Environment)

	var repo ports.UserRepository
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.InitPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL (%v). Falling back to in-memory repository.", err)
		repo = memory.NewInMemoryUserRepository()
	} else {
		log.Println("Successfully connected to PostgreSQL database.")
		defer pool.Close()
		repo = postgres.NewPostgresUserRepository(pool)
	}

	hasher := security.NewBcryptHasher(12)
	tokenManager := security.NewJWTManager(cfg.JWTSecret, cfg.TokenExpiration)
	authSvc := service.NewAuthService(repo, hasher, tokenManager)
	grpcHandler := authgrpc.NewAuthGRPCHandler(authSvc)

	// Create TCP Listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	// Create gRPC Server
	grpcServer := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, grpcHandler)

	// Enable gRPC Reflection for local debugging (grpcurl / Postman)
	reflection.Register(grpcServer)

	// Channel to capture termination signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Auth gRPC Server is listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Auth gRPC Server gracefully...")
	grpcServer.GracefulStop()
	log.Println("Auth gRPC Server stopped.")
}

