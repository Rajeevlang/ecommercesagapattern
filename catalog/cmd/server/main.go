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

	"github.com/Rajeevlang/ecommercesagapattern/catalog/config"
	cataloggrpc "github.com/Rajeevlang/ecommercesagapattern/catalog/internal/grpc"
	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/ports"
	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/repository/postgres"
	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/service"
	catalogv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/catalog/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Catalog Microservice [%s] on port %s...", cfg.Environment, cfg.GRPCPort)

	var repo ports.CatalogRepository
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to PostgreSQL database or fallback to in-memory mode
	pool, err := postgres.InitPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL database (%v). Falling back to in-memory repository.", err)
		repo = memory.NewInMemoryCatalogRepository()
	} else {
		log.Println("Successfully connected to PostgreSQL database.")
		defer pool.Close()
		repo = postgres.NewCatalogRepository(pool)
	}

	catalogSvc := service.NewCatalogService(repo)
	grpcHandler := cataloggrpc.NewGrpcHandler(catalogSvc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Catalog gRPC Server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Catalog gRPC Server gracefully...")
	grpcServer.GracefulStop()
	log.Println("Catalog gRPC Server stopped.")
}
