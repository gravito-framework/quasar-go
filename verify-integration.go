package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	// Get all quasar keys
	keys, err := client.Keys(ctx, "gravito:quasar:node:*").Result()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d Quasar nodes:\n\n", len(keys))
	
	for _, key := range keys {
		val, err := client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			continue
		}

		// Extract key info
		service := data["service"]
		language := data["language"]
		nodeID := data["id"]
		
		fmt.Printf("📍 Service: %s\n", service)
		fmt.Printf("   Language: %s\n", language)
		fmt.Printf("   Node ID: %s\n", nodeID)
		
		// CPU info
		if cpu, ok := data["cpu"].(map[string]interface{}); ok {
			fmt.Printf("   CPU: %.1f%% (system), %.2f%% (process), %v cores\n",
				cpu["system"], cpu["process"], cpu["cores"])
		}
		
		// Memory info
		if mem, ok := data["memory"].(map[string]interface{}); ok {
			if sysMem, ok := mem["system"].(map[string]interface{}); ok {
				totalGB := sysMem["total"].(float64) / 1024 / 1024 / 1024
				usedGB := sysMem["used"].(float64) / 1024 / 1024 / 1024
				fmt.Printf("   Memory: %.1f GB / %.1f GB used\n", usedGB, totalGB)
			}
		}
		
		// Queue info
		if queues, ok := data["queues"].([]interface{}); ok && len(queues) > 0 {
			fmt.Printf("   Queues:\n")
			for _, q := range queues {
				if queue, ok := q.(map[string]interface{}); ok {
					name := queue["name"]
					if size, ok := queue["size"].(map[string]interface{}); ok {
						fmt.Printf("     - %s: waiting=%v, active=%v, delayed=%v, failed=%v\n",
							name,
							size["waiting"],
							size["active"],
							size["delayed"],
							size["failed"])
					}
				}
			}
		}
		
		fmt.Println()
	}
}
