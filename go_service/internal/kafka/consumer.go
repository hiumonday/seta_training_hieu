package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"go_service/internal/events"
	"go_service/internal/redis"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	teamReader  *kafka.Reader
	assetReader *kafka.Reader
	redisService *redis.Service
}

// NewConsumer creates a new Kafka consumer for handling cache updates
func NewConsumer(brokers []string, groupID string, redisService *redis.Service) *Consumer {
	// Configure team activity reader
	teamReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          events.TeamActivityTopic,
		GroupID:        groupID + "-team",
		StartOffset:    kafka.LastOffset,
		CommitInterval: 1 * time.Second,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
	})

	// Configure asset changes reader
	assetReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          events.AssetChangesTopic,
		GroupID:        groupID + "-asset",
		StartOffset:    kafka.LastOffset,
		CommitInterval: 1 * time.Second,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
	})

	return &Consumer{
		teamReader:   teamReader,
		assetReader:  assetReader,
		redisService: redisService,
	}
}

// StartTeamEventConsumer starts consuming team events and updates Redis cache
func (c *Consumer) StartTeamEventConsumer(ctx context.Context) {
	log.Println("Starting team event consumer...")
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Team event consumer stopped")
			return
		default:
			message, err := c.teamReader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading team event message: %v", err)
				continue
			}

			var teamEvent events.TeamEvent
			if err := json.Unmarshal(message.Value, &teamEvent); err != nil {
				log.Printf("Error unmarshaling team event: %v", err)
				continue
			}

			if err := c.handleTeamEvent(ctx, &teamEvent); err != nil {
				log.Printf("Error handling team event: %v", err)
			}
		}
	}
}

// StartAssetEventConsumer starts consuming asset events and updates Redis cache
func (c *Consumer) StartAssetEventConsumer(ctx context.Context) {
	log.Println("Starting asset event consumer...")
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Asset event consumer stopped")
			return
		default:
			message, err := c.assetReader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading asset event message: %v", err)
				continue
			}

			var assetEvent events.AssetEvent
			if err := json.Unmarshal(message.Value, &assetEvent); err != nil {
				log.Printf("Error unmarshaling asset event: %v", err)
				continue
			}

			if err := c.handleAssetEvent(ctx, &assetEvent); err != nil {
				log.Printf("Error handling asset event: %v", err)
			}
		}
	}
}

// handleTeamEvent processes team events and updates Redis cache
func (c *Consumer) handleTeamEvent(ctx context.Context, event *events.TeamEvent) error {
	teamID, err := strconv.Atoi(event.TeamID)
	if err != nil {
		log.Printf("Invalid team ID: %s", event.TeamID)
		return err
	}

	switch event.EventType {
	case events.MemberAdded:
		if event.TargetUserID != "" {
			userID, err := uuid.Parse(event.TargetUserID)
			if err != nil {
				log.Printf("Invalid user ID in member added event: %s", event.TargetUserID)
				return err
			}
			return c.redisService.AddTeamMember(ctx, teamID, userID)
		}

	case events.MemberRemoved:
		if event.TargetUserID != "" {
			userID, err := uuid.Parse(event.TargetUserID)
			if err != nil {
				log.Printf("Invalid user ID in member removed event: %s", event.TargetUserID)
				return err
			}
			return c.redisService.RemoveTeamMember(ctx, teamID, userID)
		}

	case events.ManagerAdded:
		// For manager events, we might want to refresh the entire team cache
		// to ensure consistency between leaders and members
		log.Printf("Manager added to team %d, consider refreshing team cache", teamID)

	case events.ManagerRemoved:
		// Similar to manager added
		log.Printf("Manager removed from team %d, consider refreshing team cache", teamID)

	case events.TeamCreated:
		// Team created - no cache action needed initially as team will be empty
		log.Printf("Team %d created", teamID)
	}

	return nil
}

// handleAssetEvent processes asset events and updates Redis cache
func (c *Consumer) handleAssetEvent(ctx context.Context, event *events.AssetEvent) error {
	assetID, err := uuid.Parse(event.AssetID)
	if err != nil {
		log.Printf("Invalid asset ID: %s", event.AssetID)
		return err
	}

	switch event.EventType {
	// Metadata cache invalidation for updates and deletes
	case events.FolderUpdated, events.FolderDeleted:
		return c.redisService.InvalidateFolderMetadata(ctx, assetID)

	case events.NoteUpdated, events.NoteDeleted:
		return c.redisService.InvalidateNoteMetadata(ctx, assetID)

	// Access control cache updates for sharing events
	case events.FolderShared, events.NoteShared:
		if event.SharedWithUserID != nil && event.AccessLevel != nil {
			sharedWithUserID, err := uuid.Parse(*event.SharedWithUserID)
			if err != nil {
				log.Printf("Invalid shared user ID: %s", *event.SharedWithUserID)
				return err
			}
			return c.redisService.AddAssetAccess(ctx, assetID, sharedWithUserID, *event.AccessLevel)
		}

	case events.FolderUnshared, events.NoteUnshared:
		if event.SharedWithUserID != nil {
			sharedWithUserID, err := uuid.Parse(*event.SharedWithUserID)
			if err != nil {
				log.Printf("Invalid shared user ID: %s", *event.SharedWithUserID)
				return err
			}
			return c.redisService.RemoveAssetAccess(ctx, assetID, sharedWithUserID)
		}

	// For created events, we don't need to do anything special
	// as the cache will be populated when the asset is first accessed
	case events.FolderCreated, events.NoteCreated:
		log.Printf("Asset %s created", assetID)
	}

	return nil
}

// Close closes the Kafka readers
func (c *Consumer) Close() error {
	var err1, err2 error
	if c.teamReader != nil {
		err1 = c.teamReader.Close()
	}
	if c.assetReader != nil {
		err2 = c.assetReader.Close()
	}

	if err1 != nil {
		return err1
	}
	return err2
}