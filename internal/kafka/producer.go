package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
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

	// Create topic if it doesn't exist
	if err := createTopicIfNotExists(cfg.Brokers, cfg.Topic, 3, 1); err != nil {
		return nil, fmt.Errorf("failed to ensure topic exists: %w", err)
	}

	// Map compression string to kafka-go compression codec
	var compression compress.Codec
	switch cfg.Compression {
	case "none", "":
		compression = nil
	case "gzip":
		compression = compress.Gzip
	case "snappy":
		compression = compress.Snappy
	case "lz4":
		compression = compress.Lz4
	case "zstd":
		compression = compress.Zstd
	default:
		return nil, fmt.Errorf("unsupported compression: %s", cfg.Compression)
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{}, // Use hash of key for consistent partitioning
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		WriteTimeout: cfg.WriteTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
		Compression:  compression,
		MaxAttempts:  cfg.MaxAttempts,
		Async:        false, // Synchronous writes for simplicity
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

	// Get the controller broker
	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	// Connect to controller
	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	// Create topic
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: replicationFactor,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		// Check if error is "topic already exists"
		if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Title() == "Topic Already Exists" {
			return nil // Topic exists, not an error
		}
		return fmt.Errorf("failed to create topic: %w", err)
	}

	return nil
}

// Send sends a metric to Kafka.
// The metric is JSON-serialized and keyed by hostname for consistent partitioning.
// Returns an error if the message cannot be sent after all retry attempts.
func (p *Producer) Send(ctx context.Context, metric interface{}) error {
	// Serialize metric to JSON
	data, err := json.Marshal(metric)
	if err != nil {
		p.messagesFailed.Add(1)
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// Extract hostname for partitioning key
	// This ensures all metrics from the same host go to the same partition
	key := p.extractHostname(metric)

	// Create Kafka message
	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	// Send to Kafka
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.messagesFailed.Add(1)
		return fmt.Errorf("failed to write message: %w", err)
	}

	// Update metrics
	p.messagesSent.Add(1)
	p.bytesProduced.Add(uint64(len(data)))

	return nil
}

// SendBatch sends multiple metrics in a single batch.
// This is more efficient than calling Send repeatedly.
func (p *Producer) SendBatch(ctx context.Context, metrics []interface{}) error {
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

		key := p.extractHostname(metric)
		messages = append(messages, kafka.Message{
			Key:   []byte(key),
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

// extractHostname attempts to extract the hostname from a metric for use as the partition key.
// Falls back to empty string if hostname cannot be extracted.
func (p *Producer) extractHostname(metric interface{}) string {
	// Use type assertion to get hostname
	// This works with any struct that has a Hostname field
	type hostnameGetter interface {
		GetHostname() string
	}

	// Try direct interface method
	if hg, ok := metric.(hostnameGetter); ok {
		return hg.GetHostname()
	}

	// Try to access embedded BaseMetric
	// This is a bit hacky but works with our model structure
	type baseMetricHolder struct {
		BaseMetric struct {
			Hostname string `json:"hostname"`
		}
	}

	// Marshal and unmarshal to extract hostname (not efficient but works)
	// In production, you'd implement GetHostname() on all metric types
	data, err := json.Marshal(metric)
	if err != nil {
		return "unknown"
	}

	var holder baseMetricHolder
	if err := json.Unmarshal(data, &holder); err != nil {
		return "unknown"
	}

	return holder.BaseMetric.Hostname
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
