package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrewwkimm/qubeley/internal/collectors"
	"github.com/andrewwkimm/qubeley/internal/kafka"
	"github.com/andrewwkimm/qubeley/internal/models"
)

const (
	collectionInterval = 5 * time.Second
	kafkaBroker        = "localhost:9092"
	kafkaTopic         = "raw-metrics"
	shutdownGrace      = 2 * time.Second
	statsInterval      = 30 * time.Second
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Qubeley metrics collector...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	kafkaConfig := kafka.DefaultConfig(
		[]string{kafkaBroker},
		kafkaTopic,
	)
	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()
	log.Println("Kafka producer initialized")

	allCollectors := initCollectors()
	if len(allCollectors) == 0 {
		log.Fatal("No collectors could be initialized, exiting")
	}
	log.Printf("Initialized %d collectors", len(allCollectors))

	for _, c := range allCollectors {
		go runCollector(ctx, c, producer)
	}

	go reportStats(ctx, producer, statsInterval)

	log.Println("Collectors running. Press Ctrl+C to stop.")
	<-sigChan
	log.Println("Shutdown signal received, stopping collectors...")

	cancel()
	time.Sleep(shutdownGrace)

	stats := producer.Stats()
	log.Printf("Final stats: Sent=%d Failed=%d Bytes=%s",
		stats.MessagesSent,
		stats.MessagesFailed,
		formatBytes(stats.BytesProduced))

	log.Println("Qubeley stopped gracefully")
}

// initCollectors constructs all collectors, skipping those unavailable on
// the current platform and fatally exiting if a required collector fails.
func initCollectors() []collectors.Collector {
	var all []collectors.Collector

	cpu, err := collectors.NewCPUCollector(collectionInterval, true)
	if err != nil {
		log.Fatalf("Failed to create CPU collector: %v", err)
	}
	all = append(all, cpu)

	mem, err := collectors.NewMemoryCollector(collectionInterval)
	if err != nil {
		log.Fatalf("Failed to create memory collector: %v", err)
	}
	all = append(all, mem)

	temp, err := collectors.NewTemperatureCollector(collectionInterval)
	if err != nil {
		log.Fatalf("Failed to create temperature collector: %v", err)
	}
	all = append(all, temp)

	logs, err := collectors.NewLogsCollector(collectionInterval)
	if err != nil {
		if errors.Is(err, collectors.ErrNoJournal) {
			log.Printf("Skipping logs collector: %v", err)
		} else {
			log.Fatalf("Failed to create logs collector: %v", err)
		}
	} else {
		all = append(all, logs)
	}

	return all
}

// runCollector runs a single collector in a loop, sending metrics to Kafka.
func runCollector(ctx context.Context, c collectors.Collector, producer *kafka.Producer) {
	name := c.Name()
	log.Printf("Starting collector: %s (interval: %v)", name, c.Interval())

	ticker := time.NewTicker(c.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping collector: %s", name)
			return

		case <-ticker.C:
			metric, err := c.Collect()
			if err != nil {
				if isSentinel(err) {
					log.Printf("[%s] Skipping: %v", name, err)
					continue
				}
				log.Printf("[%s] Collection failed: %v", name, err)
				continue
			}

			sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if err := producer.Send(sendCtx, metric); err != nil {
				log.Printf("[%s] Failed to send to Kafka: %v", name, err)
			}
			cancel()
		}
	}
}

// isSentinel reports whether err is a known platform-availability sentinel
// that should be logged once and skipped rather than treated as a failure.
func isSentinel(err error) bool {
	return errors.Is(err, collectors.ErrNoSensors) ||
		errors.Is(err, collectors.ErrNoJournal)
}

// reportStats periodically logs producer statistics.
func reportStats(ctx context.Context, producer *kafka.Producer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := producer.Stats()
			log.Printf("Stats: Sent=%d Failed=%d Bytes=%s",
				stats.MessagesSent,
				stats.MessagesFailed,
				formatBytes(stats.BytesProduced))
		}
	}
}

// formatBytes converts bytes to a human-readable string.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Metric interface compliance is verified at compile time via producer.Send.
var _ models.Metric = (*models.CPUMetrics)(nil)
var _ models.Metric = (*models.MemoryMetrics)(nil)
var _ models.Metric = (*models.TemperatureMetrics)(nil)
var _ models.Metric = (*models.LogBatch)(nil)
