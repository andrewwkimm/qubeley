package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"github.com/andrewwkimm/qubeley/internal/models"
)

// Producer wraps a Kafka writer for sending metrics.
type Producer struct {
	writer *kafka.Writer
	config *Config

	// Metrics for monitoring
	messagesSent   atomic.Uint64
	messagesFailed atomic.Uint64
	bytesProduced  atomic.Uint64
}

// NewProducer creates a new Kafka producer with the given configuration.
// It automatically creates the topic if it doesn't exist.
func NewProducer(cfg *Config) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker address is required")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	if err := createTopicIfNotExists(cfg.Brokers, cfg.Topic, 3, 1); err != nil {
		return nil, fmt.Errorf("failed to ensure topic exists: %w", err)
	}

	var compression kafka.Compression
	switch cfg.Compression {
	case "none", "":
		compression = 0
	case "gzip":
		compression = kafka.Gzip
	case "snappy":
		compression = kafka.Snappy
	case "lz4":
		compression = kafka.Lz4
	case "zstd":
		compression = kafka.Zstd
	default:
		return nil, fmt.Errorf("unsupported compression: %s", cfg.Compression)
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		WriteTimeout: cfg.WriteTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
		Compression:  compression,
		MaxAttempts:  cfg.MaxAttempts,
		Async:        false,
	}

	return &Producer{
		writer: writer,
		config: cfg,
	}, nil
}

// createTopicIfNotExists creates a Kafka topic if it doesn't already exist.
func createTopicIfNotExists(brokers []string, topic string, partitions, replicationFactor int) error {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to broker: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: replicationFactor,
	})
	if err != nil {
		if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Title() == "Topic Already Exists" {
			return nil
		}
		return fmt.Errorf("failed to create topic: %w", err)
	}

	return nil
}

// Send sends a metric to Kafka, serialized as JSON and keyed by hostname
// for consistent partition routing.
func (p *Producer) Send(ctx context.Context, metric models.Metric) error {
	data, err := json.Marshal(metric)
	if err != nil {
		p.messagesFailed.Add(1)
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(metric.GetHostname()),
		Value: data,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.messagesFailed.Add(1)
		return fmt.Errorf("failed to write message: %w", err)
	}

	p.messagesSent.Add(1)
	p.bytesProduced.Add(uint64(len(data)))

	return nil
}

// SendBatch sends multiple metrics in a single batch.
func (p *Producer) SendBatch(ctx context.Context, metrics []models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	messages := make([]kafka.Message, 0, len(metrics))

	for _, metric := range metrics {
		data, err := json.Marshal(metric)
		if err != nil {
			p.messagesFailed.Add(1)
			return fmt.Errorf("failed to marshal metric: %w", err)
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(metric.GetHostname()),
			Value: data,
			Time:  time.Now(),
		})

		p.bytesProduced.Add(uint64(len(data)))
	}

	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		p.messagesFailed.Add(uint64(len(messages)))
		return fmt.Errorf("failed to write batch: %w", err)
	}

	p.messagesSent.Add(uint64(len(messages)))

	return nil
}

// Close gracefully shuts down the producer, flushing any pending messages.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Stats returns current producer statistics.
func (p *Producer) Stats() ProducerStats {
	return ProducerStats{
		MessagesSent:   p.messagesSent.Load(),
		MessagesFailed: p.messagesFailed.Load(),
		BytesProduced:  p.bytesProduced.Load(),
	}
}

// ProducerStats holds producer statistics for monitoring.
type ProducerStats struct {
	MessagesSent   uint64
	MessagesFailed uint64
	BytesProduced  uint64
}
