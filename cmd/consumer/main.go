package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrewwkimm/qubeley/internal/models"
	kafka "github.com/segmentio/kafka-go"
)

const (
	defaultKafkaBroker = "localhost:9092"
	defaultKafkaTopic  = "raw-metrics"
	defaultGroupID     = "qubeley-consumer"
	defaultCHURL       = "http://localhost:8123"
	defaultCHDB        = "qubeley"
	defaultCHUser      = "qubeley"
	defaultCHPassword  = "qubeley_dev"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Qubeley consumer...")

	cfg := config{
		kafkaBroker: getEnv("QUBELEY_KAFKA_BROKER", defaultKafkaBroker),
		kafkaTopic:  getEnv("QUBELEY_KAFKA_TOPIC", defaultKafkaTopic),
		groupID:     getEnv("QUBELEY_KAFKA_GROUP_ID", defaultGroupID),
		chURL:       getEnv("QUBELEY_CLICKHOUSE_URL", defaultCHURL),
		chDB:        getEnv("QUBELEY_CLICKHOUSE_DB", defaultCHDB),
		chUser:      getEnv("QUBELEY_CLICKHOUSE_USER", defaultCHUser),
		chPassword:  getEnv("QUBELEY_CLICKHOUSE_PASSWORD", defaultCHPassword),
	}

	ch, err := newClickHouseClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer ch.close()
	log.Printf("ClickHouse client ready (%s/%s)", cfg.chURL, cfg.chDB)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{cfg.kafkaBroker},
		Topic:          cfg.kafkaTopic,
		GroupID:        cfg.groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	defer reader.Close()
	log.Printf("Kafka reader ready (broker: %s, topic: %s, group: %s)",
		cfg.kafkaBroker, cfg.kafkaTopic, cfg.groupID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping consumer...")
		cancel()
	}()

	log.Println("Consumer running. Press Ctrl+C to stop.")
	if err := consume(ctx, reader, ch); err != nil && err != context.Canceled {
		log.Fatalf("Consumer error: %v", err)
	}
	log.Println("Qubeley consumer stopped gracefully.")
}

// consume reads messages from Kafka and routes them to ClickHouse.
func consume(ctx context.Context, reader *kafka.Reader, ch *clickHouseClient) error {
	var processed, failed uint64

	statsTicker := time.NewTicker(30 * time.Second)
	defer statsTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-statsTicker.C:
			log.Printf("Stats: processed=%d failed=%d", processed, failed)
		default:
		}

		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("Failed to read message: %v", err)
			failed++
			continue
		}

		if err := route(ctx, msg.Value, ch); err != nil {
			log.Printf("Failed to route message: %v", err)
			failed++
			continue
		}

		processed++
	}
}

// route unmarshals the metric_type from a raw payload, writes to raw_metrics
// for the audit log, then dispatches to the correct structured insert.
func route(ctx context.Context, payload []byte, ch *clickHouseClient) error {
	var base models.BaseMetric
	if err := json.Unmarshal(payload, &base); err != nil {
		return fmt.Errorf("failed to unmarshal base metric: %w", err)
	}

	// Always write to raw_metrics audit table.
	if err := ch.insertRaw(ctx, &base, string(payload)); err != nil {
		log.Printf("Warning: failed to insert raw metric: %v", err)
	}

	switch base.MetricType {
	case "cpu":
		var m models.CPUMetrics
		if err := json.Unmarshal(payload, &m); err != nil {
			return fmt.Errorf("failed to unmarshal cpu metric: %w", err)
		}
		return ch.insertCPU(ctx, &m)

	case "memory":
		var m models.MemoryMetrics
		if err := json.Unmarshal(payload, &m); err != nil {
			return fmt.Errorf("failed to unmarshal memory metric: %w", err)
		}
		return ch.insertMemory(ctx, &m)

	case "temperature":
		var m models.TemperatureMetrics
		if err := json.Unmarshal(payload, &m); err != nil {
			return fmt.Errorf("failed to unmarshal temperature metric: %w", err)
		}
		return ch.insertTemperature(ctx, &m)

	case "logs":
		var m models.LogBatch
		if err := json.Unmarshal(payload, &m); err != nil {
			return fmt.Errorf("failed to unmarshal log batch: %w", err)
		}
		return ch.insertLogs(ctx, &m)

	default:
		return fmt.Errorf("unknown metric_type: %q", base.MetricType)
	}
}

type config struct {
	kafkaBroker string
	kafkaTopic  string
	groupID     string
	chURL       string
	chDB        string
	chUser      string
	chPassword  string
}