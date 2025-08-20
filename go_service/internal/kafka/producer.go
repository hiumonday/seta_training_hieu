package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"go_service/internal/events"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	teamWriter  *kafka.Writer
	assetWriter *kafka.Writer
}

// NewProducer creates a new Kafka producer with writers for different topics
func NewProducer(brokers []string) *Producer {
	// Configure team activity writer
	teamWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        events.TeamActivityTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	// Configure asset changes writer
	assetWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        events.AssetChangesTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	return &Producer{
		teamWriter:  teamWriter,
		assetWriter: assetWriter,
	}
}

// PublishTeamEvent publishes a team event to the team.activity topic
func (p *Producer) PublishTeamEvent(ctx context.Context, event *events.TeamEvent) error {
	value, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal team event: %v", err)
		return err
	}

	message := kafka.Message{
		Key:   []byte(event.TeamID),
		Value: value,
		Time:  event.Timestamp,
	}

	err = p.teamWriter.WriteMessages(ctx, message)
	if err != nil {
		log.Printf("Failed to publish team event: %v", err)
		return err
	}

	log.Printf("Published team event: %s for team %s", event.EventType, event.TeamID)
	return nil
}

// PublishAssetEvent publishes an asset event to the asset.changes topic
func (p *Producer) PublishAssetEvent(ctx context.Context, event *events.AssetEvent) error {
	value, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal asset event: %v", err)
		return err
	}

	message := kafka.Message{
		Key:   []byte(event.AssetID),
		Value: value,
		Time:  event.Timestamp,
	}

	err = p.assetWriter.WriteMessages(ctx, message)
	if err != nil {
		log.Printf("Failed to publish asset event: %v", err)
		return err
	}

	log.Printf("Published asset event: %s for %s %s", event.EventType, event.AssetType, event.AssetID)
	return nil
}

// Close closes the Kafka writers
func (p *Producer) Close() error {
	var err1, err2 error
	if p.teamWriter != nil {
		err1 = p.teamWriter.Close()
	}
	if p.assetWriter != nil {
		err2 = p.assetWriter.Close()
	}

	if err1 != nil {
		return err1
	}
	return err2
}