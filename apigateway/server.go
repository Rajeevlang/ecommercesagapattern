package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Rajeevlang/ecommercesagapattern/apigateway/graph"
	gatewaymw "github.com/Rajeevlang/ecommercesagapattern/apigateway/middleware"
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	authv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/auth/v1"
	catalogv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/catalog/v1"
	orderv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/order/v1"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Service addresses from environment
	authAddr := os.Getenv("AUTH_SVC_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50051"
	}
	accountAddr := os.Getenv("ACCOUNT_SVC_ADDR")
	if accountAddr == "" {
		accountAddr = "localhost:50052"
	}
	orderAddr := os.Getenv("ORDER_SVC_ADDR")
	if orderAddr == "" {
		orderAddr = "localhost:50053"
	}
	catalogAddr := os.Getenv("CATALOG_SVC_ADDR")
	if catalogAddr == "" {
		catalogAddr = "localhost:50054"
	}

	log.Println("Establishing gRPC client connections...")

	// 1. Dial Auth Service
	authConn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Auth Service at %s: %v", authAddr, err)
	}
	defer authConn.Close()
	authClient := authv1.NewAuthServiceClient(authConn)

	// 2. Dial Account Service
	accountConn, err := grpc.NewClient(accountAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Account Service at %s: %v", accountAddr, err)
	}
	defer accountConn.Close()
	accountClient := accountv1.NewAccountServiceClient(accountConn)

	// 3. Dial Catalog Service
	catalogConn, err := grpc.NewClient(catalogAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Catalog Service at %s: %v", catalogAddr, err)
	}
	defer catalogConn.Close()
	catalogClient := catalogv1.NewCatalogServiceClient(catalogConn)

	// 4. Dial Order Service
	orderConn, err := grpc.NewClient(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Order Service at %s: %v", orderAddr, err)
	}
	defer orderConn.Close()
	orderClient := orderv1.NewOrderServiceClient(orderConn)

	log.Println("gRPC client connections initialized successfully.")

	// Configure GraphQL resolver dependencies
	resolver := &graph.Resolver{
		AuthClient:    authClient,
		AccountClient: accountClient,
		CatalogClient: catalogClient,
		OrderClient:   orderClient,
	}

	// Initialize GraphQL handler
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Setup Router using Chi
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Apply Custom JWT verification middleware to authenticate requests before GraphQL execution
	r.Use(gatewaymw.AuthMiddleware(authClient))

	// Register Routes
	r.Handle("/", playground.Handler("GraphQL playground", "/query"))
	r.Handle("/query", srv)

	log.Printf("Connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
