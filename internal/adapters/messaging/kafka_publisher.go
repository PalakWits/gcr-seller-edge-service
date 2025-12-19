package messaging

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"adapter/internal/ports"
)

type KafkaConfig struct {
	Brokers string
}

// KafkaPublisher implements ports.EventPublisher using kafka-go.
type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(cfg KafkaConfig) (ports.EventPublisher, error) {
	brokers := strings.Split(cfg.Brokers, ",")
	if len(brokers) == 0 {
		return nil, fmt.Errorf("no kafka brokers configured")
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}

	return &KafkaPublisher{writer: writer}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, topic string, key, value []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msg := kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish kafka message: %w", err)
	}
	return nil
}
