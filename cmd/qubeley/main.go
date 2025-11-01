package main

import (
	"fmt"
	"log"
	"time"

	"github.com/andrewwkimm/qubeley/internal/collectors"
)

func main() {
	// Initialize CPU collector with 5-second interval, per-core metrics enabled
	cpuCollector, err := collectors.NewCPUCollector(5*time.Second, true)
	if err != nil {
		log.Fatalf("Failed to create CPU collector: %v", err)
	}

	// Collect metrics
	metrics, err := cpuCollector.Collect()
	if err != nil {
		log.Fatalf("Error collecting CPU metrics: %v", err)
	}

	// Print the collected metrics
	fmt.Printf("Collected metrics: %+v\n", metrics)
}
