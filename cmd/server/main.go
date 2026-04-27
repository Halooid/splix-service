package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"time"

	splixv1 "github.com/halooid/backend/splix-service/gen/go/splix/v1"
	"github.com/halooid/backend/splix-service/internal/db"
	"github.com/halooid/backend/splix-service/internal/handler"
	"github.com/halooid/backend/splix-service/internal/service"
	"github.com/halooid/backend/splix-service/internal/db/migration"
	"github.com/halooid/backend/go-shared/auth"
	"github.com/halooid/backend/go-shared/logging"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)


func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	jwksURL := os.Getenv("JWKS_URL")
	if jwksURL == "" {
		log.Fatal("JWKS_URL is required")
	}

	conn, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Run migrations
	runMigrations(conn)

	ctx := context.Background()
	validator, err := auth.NewValidator(ctx, jwksURL, 15*time.Minute)
	if err != nil {
		log.Fatalf("failed to initialize auth validator: %v", err)
	}

	queries := db.New(conn)
	splixService := service.NewSplixService(queries)
	splixHandler := handler.NewSplixHandler(splixService)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(logging.UnaryServerInterceptor()),
		grpc.ChainUnaryInterceptor(auth.UnaryServerInterceptor(validator)),
	)
	splixv1.RegisterUserServiceServer(grpcServer, splixHandler)
	splixv1.RegisterGroupServiceServer(grpcServer, splixHandler)
	splixv1.RegisterExpenseServiceServer(grpcServer, splixHandler)

	reflection.Register(grpcServer)

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("Splix Service starting on gRPC port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("failed to create migration driver: %v", err)
	}

	sourceDriver, err := iofs.New(migration.MigrationFS, ".")
	if err != nil {
		log.Fatalf("failed to create source driver: %v", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs", sourceDriver,
		"postgres", driver)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("Migrations completed successfully")
}
