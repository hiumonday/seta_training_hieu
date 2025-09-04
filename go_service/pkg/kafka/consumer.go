package kafka

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type EventHandler func(event TeamEvent) error

type Consumer struct {
	consumer *kafka.Consumer
	handlers map[string][]EventHandler
	topic    string
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(bootstrapServers, username, password, topic string) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		// User-specific properties
		"bootstrap.servers": bootstrapServers,
		"sasl.username":     username,
		"sasl.password":     password,

		// Fixed properties
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"group.id":          "team-consumer",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %s", err)
	}

	err = c.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	return &Consumer{
		consumer: c,
		handlers: make(map[string][]EventHandler),
		topic:    topic,
	}, nil
}

// RegisterHandler registers a handler for a specific event type
func (c *Consumer) RegisterHandler(eventType string, handler EventHandler) {
	if c.handlers[eventType] == nil {
		c.handlers[eventType] = []EventHandler{}
	}
	c.handlers[eventType] = append(c.handlers[eventType], handler)
}

// Start starts consuming messages
func (c *Consumer) Start() {
	// Set up a channel for handling Ctrl-C, etc
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Process messages
	run := true
	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			ev, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Errors are informational and automatically handled by the consumer
				continue
			}

			fmt.Printf("Consumed event from topic %s: key = %-10s\n",
				*ev.TopicPartition.Topic, string(ev.Key))

			// Parse the TeamEvent from the message
			var event TeamEvent
			if err := json.Unmarshal(ev.Value, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v\n", err)
				continue
			}

			// Process event with registered handlers
			if handlers, ok := c.handlers[event.EventType]; ok {
				for _, handler := range handlers {
					if err := handler(event); err != nil {
						log.Printf("Error handling event %s: %v\n", event.EventType, err)
					}
				}
			}
		}
	}

	c.consumer.Close()
}

// Close the consumer
func (c *Consumer) Close() {
	c.consumer.Close()
}
