package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrewwkimm/qubeley/internal/collectors"
	"github.com/andrewwkimm/qubeley/internal/kafka"
)

func main() {
	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting Qubeley metrics collector...")

	// Create context that listens for termination signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize Kafka producer (automatically creates topic)
	kafkaConfig := kafka.DefaultConfig(
		[]string{"kafka:29092"}, // Use service name from docker-compose
		"raw-metrics",
	)
	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		log.Fatalf("❌ Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()
	log.Println("✅ Kafka producer initialized")

	// Create collectors
	cpuCollector, err := collectors.NewCPUCollector(5*time.Second, true)
	if err != nil {
		log.Fatalf("❌ Failed to create CPU collector: %v", err)
	}

	allCollectors := []collectors.Collector{
		cpuCollector,
		// Add more collectors here as you build them:
		// memoryCollector,
		// temperatureCollector,
	}

	log.Printf("✅ Initialized %d collectors\n", len(allCollectors))

	// Start collectors in separate goroutines
	for _, collector := range allCollectors {
		go runCollector(ctx, collector, producer)
	}

	// Start stats reporter
	go reportStats(ctx, producer, 30*time.Second)

	log.Println("📊 Collectors running. Press Ctrl+C to stop.")

	// Wait for termination signal
	<-sigChan
	log.Println("\n🛑 Shutdown signal received, stopping collectors...")

	// Cancel context to stop all goroutines
	cancel()

	// Give goroutines time to finish
	time.Sleep(2 * time.Second)

	// Print final stats
	stats := producer.Stats()
	log.Printf("📈 Final stats: Sent=%d Failed=%d Bytes=%d\n",
		stats.MessagesSent, stats.MessagesFailed, stats.BytesProduced)

	log.Println("👋 Qubeley stopped gracefully")
}

// runCollector runs a single collector in a loop, sending metrics to Kafka.
func runCollector(ctx context.Context, collector collectors.Collector, producer *kafka.Producer) {
	name := collector.Name()
	interval := collector.Interval()

	log.Printf("▶️  Starting collector: %s (interval: %v)", name, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("⏹️  Stopping collector: %s", name)
			return

		case <-ticker.C:
			// Collect metrics
			metric, err := collector.Collect()
			if err != nil {
				log.Printf("⚠️  [%s] Collection failed: %v", name, err)
				continue
			}

			// Send to Kafka with timeout
			sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if err := producer.Send(sendCtx, metric); err != nil {
				log.Printf("⚠️  [%s] Failed to send to Kafka: %v", name, err)
			} else {
				log.Printf("✓ [%s] Metric sent successfully", name)
			}
			cancel()
		}
	}
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
			log.Printf("📊 Stats: Sent=%d Failed=%d Bytes=%s",
				stats.MessagesSent,
				stats.MessagesFailed,
				formatBytes(stats.BytesProduced))
		}
	}
}

// formatBytes converts bytes to human-readable format.
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
