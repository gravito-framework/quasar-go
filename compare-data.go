package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	// Get both nodes
	keys, _ := client.Keys(ctx, "gravito:quasar:node:*").Result()
	
	for _, key := range keys {
		val, _ := client.Get(ctx, key).Result()
		
		var data map[string]interface{}
		json.Unmarshal([]byte(val), &data)
		
		service := data["service"]
		language := data["language"]
		
		fmt.Printf("=== %s (%s) ===\n", service, language)
		
		// Show CPU data structure
		if cpu, ok := data["cpu"].(map[string]interface{}); ok {
			fmt.Printf("CPU data:\n")
			cpuJSON, _ := json.MarshalIndent(cpu, "  ", "  ")
			fmt.Printf("  %s\n", string(cpuJSON))
		}
		
		fmt.Println()
	}
}
