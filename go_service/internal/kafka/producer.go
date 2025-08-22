package kafka

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/google/uuid"
)

// TeamEvent represents a team activity event
type TeamEvent struct {
	EventType    string    `json:"eventType"`
	TeamID       uint64    `json:"teamId"`
	PerformedBy  uuid.UUID `json:"performedBy"`
	TargetUserID uuid.UUID `json:"targetUserId,omitempty"`
	Timestamp    string    `json:"timestamp"`
}

// EventType constants
const (
	EventTeamCreated    = "TEAM_CREATED"
	EventMemberAdded    = "MEMBER_ADDED"
	EventMemberRemoved  = "MEMBER_REMOVED"
	EventManagerAdded   = "MANAGER_ADDED"
	EventManagerRemoved = "MANAGER_REMOVED"
)

// Producer encapsulates a Kafka producer
type Producer struct {
	producer *kafka.Producer
	topic    string
}

// NewProducer creates a new Kafka producer
func NewProducer(bootstrapServers, username, password, topic string) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		// User-specific properties
		"bootstrap.servers": bootstrapServers,
		"sasl.username":     username,
		"sasl.password":     password,

		// Fixed properties
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"acks":              "all",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %s", err)
	}

	// Go-routine to handle message delivery reports
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Failed to deliver message: %v\n", ev.TopicPartition)
				} else {
					log.Printf("Produced event to topic %s: key = %-10s\n",
						*ev.TopicPartition.Topic, string(ev.Key))
				}
			}
		}
	}()

	return &Producer{
		producer: p,
		topic:    topic,
	}, nil
}

// SendTeamEvent sends a team event to the Kafka topic
func (p *Producer) SendTeamEvent(eventType string, teamID uint64, performedBy, targetUserID uuid.UUID) error {
	event := TeamEvent{
		EventType:    eventType,
		TeamID:       teamID,
		PerformedBy:  performedBy,
		TargetUserID: targetUserID,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Use the team ID as the key for partitioning
	key := fmt.Sprintf("team-%d", teamID)

	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          eventJSON,
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// Close closes the producer
func (p *Producer) Close() {
	p.producer.Flush(15 * 1000) // 15 seconds timeout
	p.producer.Close()
}
