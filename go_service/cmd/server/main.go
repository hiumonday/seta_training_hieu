package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go_service/internal/database"
	"go_service/internal/kafka"
	"go_service/internal/redis"
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

	// Initialize Kafka producer (optional - can be nil for testing)
	var kafkaProducer *kafka.Producer
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers != "" {
		brokers := strings.Split(kafkaBrokers, ",")
		kafkaProducer = kafka.NewProducer(brokers)
		log.Printf("Kafka producer initialized with brokers: %v", brokers)
	} else {
		log.Println("KAFKA_BROKERS not set, Kafka events will be disabled")
	}

	// Initialize Redis service (optional - can be nil for testing)
	var redisService *redis.Service
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default Redis address
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisService = redis.NewService(redisAddr, redisPassword, 0)
	if redisService == nil {
		log.Println("Redis connection failed, caching will be disabled")
	} else {
		log.Printf("Redis service initialized with addr: %s", redisAddr)
	}

	// Setup Gin router
	r := gin.Default()
	// middleware.SetupPrometheus(r)
	// r.Use(middleware.LoggerMiddleware())
	router.SetupRouter(r, db, kafkaProducer, redisService)

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

	// Cleanup Kafka and Redis connections
	if kafkaProducer != nil {
		if err := kafkaProducer.Close(); err != nil {
			log.Printf("Error closing Kafka producer: %v", err)
		}
	}
	if redisService != nil {
		if err := redisService.Close(); err != nil {
			log.Printf("Error closing Redis service: %v", err)
		}
	}

	// Attempt to do a graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
