package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go_service/internal/kafka"
	"go_service/internal/redisclient"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	redis_client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // Use default DB
		Protocol: 2, // Connection protocol
	})
	teamCache := redisclient.NewTeamCache(redis_client)

	consumer, err := kafka.NewConsumer(
		os.Getenv("BOOTSTRAP_HOST"), // bootstrap servers
		os.Getenv("KAFKA_USERNAME"), // username
		os.Getenv("KAFKA_PASSWORD"), // password
		"team.activity",             // topic
	)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()

	// Register event handlers
	consumer.RegisterHandler(kafka.EventMemberAdded, createMemberAddedHandler(teamCache))
	consumer.RegisterHandler(kafka.EventMemberRemoved, createMemberRemovedHandler(teamCache))

	// Start consuming events
	fmt.Println("Starting to consume team events...")
	consumer.Start()
}

// Factory functions that return handlers
func createMemberAddedHandler(teamCache *redisclient.TeamCache) func(kafka.TeamEvent) error {
	return func(event kafka.TeamEvent) error {
		fmt.Printf("[%s] Member added: TeamID=%d, Member=%s, AddedBy=%s\n",
			event.Timestamp, event.TeamID, event.TargetUserID, event.PerformedBy)

		// Update Redis cache
		if teamCache != nil {
			ctx := context.Background()

			memberID := event.TargetUserID

			// Add member to Redis cache
			if err := teamCache.AddMember(ctx, event.TeamID, memberID); err != nil {
				log.Printf("Error updating Redis cache for member addition: %v", err)
				return err
			}

			log.Printf("Successfully updated Redis cache: added member %s to team %d",
				event.TargetUserID, event.TeamID)
		}

		return nil
	}
}

func createMemberRemovedHandler(teamCache *redisclient.TeamCache) func(kafka.TeamEvent) error {
	return func(event kafka.TeamEvent) error {
		fmt.Printf("[%s] Member removed: TeamID=%d, Member=%s, RemovedBy=%s\n",
			event.Timestamp, event.TeamID, event.TargetUserID, event.PerformedBy)

		// Update Redis cache
		if teamCache != nil {
			ctx := context.Background()

			memberID := event.TargetUserID

			// Remove member from Redis cache
			if err := teamCache.RemoveMember(ctx, event.TeamID, memberID); err != nil {
				log.Printf("Error updating Redis cache for member removal: %v", err)
				return err
			}

			log.Printf("Successfully updated Redis cache: removed member %s from team %d",
				event.TargetUserID, event.TeamID)
		}

		return nil
	}
}
