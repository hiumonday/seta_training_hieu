package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go_service/internal/kafka"
	"go_service/internal/redis"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize Redis service
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisService := redis.NewService(redisAddr, redisPassword, 0)
	if redisService == nil {
		log.Fatal("Failed to connect to Redis")
	}

	// Initialize Kafka consumer
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	brokers := strings.Split(kafkaBrokers, ",")
	consumer := kafka.NewConsumer(brokers, "cache-updater", redisService)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumers in goroutines
	go consumer.StartTeamEventConsumer(ctx)
	go consumer.StartAssetEventConsumer(ctx)

	log.Println("Kafka consumer started. Press Ctrl+C to exit.")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down consumer...")
	cancel()

	// Cleanup
	consumer.Close()
	redisService.Close()

	log.Println("Consumer exited")
}