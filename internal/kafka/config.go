// Package kafka provides Kafka producer and consumer implementations.
package kafka

import (
	"time"
)

// Config holds Kafka connection and behavior settings.
type Config struct {
	// Brokers is the list of Kafka broker addresses
	Brokers []string

	// Topic is the Kafka topic to produce messages to
	Topic string

	// BatchSize is the maximum number of messages to batch before sending
	// Set to 1 for immediate sending (lower latency)
	// Set higher for better throughput
	BatchSize int

	// BatchTimeout is how long to wait before sending a partial batch
	BatchTimeout time.Duration

	// WriteTimeout is the maximum time to wait for a write to complete
	WriteTimeout time.Duration

	// ReadTimeout is the maximum time to wait for a read to complete
	ReadTimeout time.Duration

	// RequiredAcks defines the number of acknowledgements required
	// -1 = all replicas (safest, slowest)
	//  0 = no acknowledgement (fastest, can lose data)
	//  1 = leader only (balanced)
	RequiredAcks int

	// Compression algorithm to use (none, gzip, snappy, lz4, zstd)
	Compression string

	// MaxAttempts is the number of times to retry sending a message
	MaxAttempts int
}

// DefaultConfig returns a configuration with sensible defaults for development.
func DefaultConfig(brokers []string, topic string) *Config {
	return &Config{
		Brokers:      brokers,
		Topic:        topic,
		BatchSize:    100,              // Batch up to 100 messages
		BatchTimeout: 1 * time.Second,  // Or send after 1 second
		WriteTimeout: 10 * time.Second, // Fail write after 10s
		ReadTimeout:  10 * time.Second, // Fail read after 10s
		RequiredAcks: 1,                // Wait for leader ack (balanced)
		Compression:  "snappy",         // Fast compression
		MaxAttempts:  3,                // Retry up to 3 times
	}
}

// ProductionConfig returns a configuration optimized for production use.
func ProductionConfig(brokers []string, topic string) *Config {
	return &Config{
		Brokers:      brokers,
		Topic:        topic,
		BatchSize:    1000, // Larger batches
		BatchTimeout: 100 * time.Millisecond,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
		RequiredAcks: -1,     // All replicas (safest)
		Compression:  "zstd", // Best compression
		MaxAttempts:  5,
	}
}
