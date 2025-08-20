package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_service/internal/database"
	"go_service/internal/kafka"

	// "go_service/internal/logger"
	// "go_service/internal/middleware"
	"go_service/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// logger.InitLogger()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	dsn := database.CreateDSN()
	log.Printf("dsn: %s", dsn)
	db, err := database.Connect(dsn)
	if err != nil {
		log.Fatalf("Fail to connect DB: %v", err)
	}

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(
		os.Getenv("BOOTSTRAP_HOST"), // bootstrap servers
		os.Getenv("KAFKA_USERNAME"), // username
		os.Getenv("KAFKA_PASSWORD"), // password
		"team.activity",             // topic
	)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Setup Gin router
	r := gin.Default()
	// middleware.SetupPrometheus(r)
	// r.Use(middleware.LoggerMiddleware())
	router.SetupRouter(r, db, kafkaProducer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create a new HTTP server with the Gin router
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	//add graceful shutdown
	// Start the server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	// Listen for SIGINT and SIGTERM signals
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to do a graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
