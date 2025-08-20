package main

import (
	"fmt"
	"log"
	"os"

	"go_service/internal/kafka"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	// Initialize Kafka consumer
	fmt.Println(os.Getenv("KAFKA_USERNAME"))
	fmt.Println(os.Getenv("KAFKA_PASSWORD"))
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
	// consumer.RegisterHandler(kafka.EventTeamCreated, handleTeamCreated)
	consumer.RegisterHandler(kafka.EventMemberAdded, handleMemberAdded)
	consumer.RegisterHandler(kafka.EventMemberRemoved, handleMemberRemoved)
	// consumer.RegisterHandler(kafka.EventManagerAdded, handleManagerAdded)
	// consumer.RegisterHandler(kafka.EventManagerRemoved, handleManagerRemoved)

	// Start consuming events
	fmt.Println("Starting to consume team events...")
	consumer.Start()
}

// func handleTeamCreated(event kafka.TeamEvent) error {
// 	fmt.Printf("[%s] Team created: TeamID=%d, Creator=%s\n",
// 		event.Timestamp, event.TeamID, event.PerformedBy)
//
// 	return nil
// }

func handleMemberAdded(event kafka.TeamEvent) error {
	fmt.Printf("[%s] Member added: TeamID=%d, Member=%s, AddedBy=%s\n",
		event.Timestamp, event.TeamID, event.TargetUserID, event.PerformedBy)
	// Could save to database, send notifications, etc.
	return nil
}

func handleMemberRemoved(event kafka.TeamEvent) error {
	fmt.Printf("[%s] Member removed: TeamID=%d, Member=%s, RemovedBy=%s\n",
		event.Timestamp, event.TeamID, event.TargetUserID, event.PerformedBy)
	// Could save to database, send notifications, etc.
	return nil
}

// func handleManagerAdded(event kafka.TeamEvent) error {
// 	fmt.Printf("[%s] Manager added: TeamID=%d, Manager=%s, AddedBy=%s\n",
// 		event.Timestamp, event.TeamID, event.TargetUserID, event.PerformedBy)
//
// 	return nil
// }

// func handleManagerRemoved(event kafka.TeamEvent) error {
// 	fmt.Printf("[%s] Manager removed: TeamID=%d, Manager=%s, RemovedBy=%s\n",
// 		event.Timestamp, event.TeamID, event.TargetUserID, event.PerformedBy)
//
// 	return nil
// }
