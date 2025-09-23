package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cuemby/gor/internal/cable"
	"github.com/cuemby/gor/internal/cache"
	"github.com/cuemby/gor/internal/queue"
)

func main() {
	fmt.Println("\n🚀 Gor Framework - Solid Trifecta Demo")
	fmt.Println("=======================================\n")

	// Demo Queue
	demoQueue()

	// Demo Cache
	demoCache()

	// Demo Cable (PubSub)
	demoCable()

	fmt.Println("\n✅ All Solid Trifecta components demonstrated successfully!")
}

func demoQueue() {
	fmt.Println("📬 Testing Solid Queue (Database-backed Job Processing)")
	fmt.Println("--------------------------------------------------------")

	// Create a new queue
	sq, err := queue.NewSolidQueue("demo_queue.db", 3)
	if err != nil {
		log.Fatal("Failed to create queue:", err)
	}

	// Register job handlers
	sq.RegisterHandler("email", func(ctx *queue.JobContext) error {
		fmt.Printf("  📧 Sending email (Job %s, Attempt %d): %v\n", ctx.ID, ctx.Attempt, ctx.Payload)
		time.Sleep(500 * time.Millisecond) // Simulate work
		return nil
	})

	sq.RegisterHandler("report", func(ctx *queue.JobContext) error {
		fmt.Printf("  📊 Generating report (Job %s): %v\n", ctx.ID, ctx.Payload)
		time.Sleep(1 * time.Second) // Simulate work
		return nil
	})

	sq.RegisterHandler("failing_job", func(ctx *queue.JobContext) error {
		fmt.Printf("  ❌ Failing job (Job %s, Attempt %d)\n", ctx.ID, ctx.Attempt)
		return fmt.Errorf("intentional failure for demo")
	})

	// Start the queue workers
	ctx := context.Background()
	if err := sq.Start(ctx); err != nil {
		log.Fatal("Failed to start queue:", err)
	}

	// Enqueue some jobs
	fmt.Println("  Enqueuing jobs...")

	// Immediate job
	emailJob := &queue.Job{
		Handler: "email",
		Payload: map[string]string{
			"to":      "user@example.com",
			"subject": "Welcome to Gor!",
		},
	}
	if err := sq.Enqueue(emailJob); err != nil {
		log.Printf("Failed to enqueue email job: %v", err)
	}
	fmt.Printf("  ✓ Enqueued email job (ID: %s)\n", emailJob.ID)

	// Scheduled job
	reportJob := &queue.Job{
		Handler: "report",
		Payload: map[string]interface{}{
			"type":   "daily",
			"format": "PDF",
		},
	}
	if err := sq.EnqueueIn(reportJob, 2*time.Second); err != nil {
		log.Printf("Failed to enqueue report job: %v", err)
	}
	fmt.Printf("  ✓ Scheduled report job for 2 seconds from now (ID: %s)\n", reportJob.ID)

	// Job that will retry
	failingJob := &queue.Job{
		Handler:     "failing_job",
		MaxAttempts: 2,
	}
	if err := sq.Enqueue(failingJob); err != nil {
		log.Printf("Failed to enqueue failing job: %v", err)
	}
	fmt.Printf("  ✓ Enqueued job that will fail and retry (ID: %s)\n", failingJob.ID)

	// Let jobs process
	fmt.Println("  Processing jobs...")
	time.Sleep(5 * time.Second)

	// Get stats
	stats, err := sq.GetStats()
	if err == nil {
		fmt.Println("\n  Queue Statistics:")
		for status, count := range stats["jobs_by_status"].(map[string]int) {
			fmt.Printf("    %s: %d\n", status, count)
		}
	}

	// Stop the queue
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sq.Stop(shutdownCtx)

	fmt.Println("")
}

func demoCache() {
	fmt.Println("💾 Testing Solid Cache (Database-backed Caching)")
	fmt.Println("-------------------------------------------------")

	// Create a new cache with 10MB memory limit
	sc, err := cache.NewSolidCache("demo_cache.db", 10)
	if err != nil {
		log.Fatal("Failed to create cache:", err)
	}
	defer sc.Close()

	// Set some values
	fmt.Println("  Setting cache values...")

	// Simple value with TTL
	if err := sc.Set("user:1", map[string]interface{}{
		"id":    1,
		"name":  "John Doe",
		"email": "john@example.com",
	}, 10*time.Second); err != nil {
		log.Printf("Failed to set user:1: %v", err)
	}
	fmt.Println("  ✓ Set user:1 with 10 second TTL")

	// Permanent value (no TTL)
	if err := sc.Set("config:app", map[string]string{
		"name":    "Gor Demo",
		"version": "1.0.0",
	}, 0); err != nil {
		log.Printf("Failed to set config:app: %v", err)
	}
	fmt.Println("  ✓ Set config:app with no expiration")

	// Counter
	if err := sc.Set("counter:views", 0, 0); err != nil {
		log.Printf("Failed to set counter: %v", err)
	}
	fmt.Println("  ✓ Set counter:views to 0")

	// Get values
	fmt.Println("\n  Getting cache values...")

	if user, err := sc.Get("user:1"); err == nil && user != nil {
		fmt.Printf("  ✓ Got user:1: %v\n", user)
	}

	if config, err := sc.Get("config:app"); err == nil && config != nil {
		fmt.Printf("  ✓ Got config:app: %v\n", config)
	}

	// Increment counter
	fmt.Println("\n  Testing atomic operations...")
	for i := 0; i < 5; i++ {
		if val, err := sc.Increment("counter:views", 1); err == nil {
			fmt.Printf("  ✓ Incremented counter to: %d\n", val)
		}
	}

	// Test cache miss
	fmt.Println("\n  Testing cache miss...")
	if val, err := sc.Get("nonexistent"); err == nil && val == nil {
		fmt.Println("  ✓ Cache miss handled correctly")
	}

	// Test Fetch pattern (compute if missing)
	fmt.Println("\n  Testing fetch pattern...")
	result, err := sc.Fetch("computed:value", 30*time.Second, func() (interface{}, error) {
		fmt.Println("  ⚙️  Computing expensive value...")
		time.Sleep(500 * time.Millisecond) // Simulate expensive computation
		return map[string]interface{}{
			"computed_at": time.Now().Format(time.RFC3339),
			"value":       42,
		}, nil
	})
	if err == nil {
		fmt.Printf("  ✓ Fetched/computed value: %v\n", result)
	}

	// Second fetch should come from cache
	result2, err := sc.Fetch("computed:value", 30*time.Second, func() (interface{}, error) {
		fmt.Println("  ⚙️  This shouldn't be called!")
		return nil, nil
	})
	if err == nil {
		fmt.Printf("  ✓ Fetched from cache (no computation): %v\n", result2)
	}

	// Get stats
	stats, err := sc.GetStats()
	if err == nil {
		fmt.Println("\n  Cache Statistics:")
		fmt.Printf("    Memory entries: %v\n", stats["memory_entries"])
		fmt.Printf("    Memory size: %.2f MB\n", stats["memory_size_mb"])
		fmt.Printf("    DB entries: %v\n", stats["db_entries"])
		fmt.Printf("    DB size: %.2f MB\n", stats["db_size_mb"])
		fmt.Printf("    Total hits: %v\n", stats["total_hits"])
	}

	fmt.Println("")
}

func demoCable() {
	fmt.Println("📡 Testing Solid Cable (Database-backed Pub/Sub)")
	fmt.Println("-------------------------------------------------")

	// Create a new cable instance
	sc, err := cable.NewSolidCable("demo_cable.db")
	if err != nil {
		log.Fatal("Failed to create cable:", err)
	}
	defer sc.Close()

	ctx := context.Background()

	// Subscribe to channels
	fmt.Println("  Creating subscriptions...")

	// User events subscription
	userSub, err := sc.Subscribe(ctx, "user_events", func(ctx context.Context, msg *cable.Message) error {
		fmt.Printf("  📨 [user_events] Received: %v\n", msg.Data)
		return nil
	})
	if err != nil {
		log.Printf("Failed to subscribe to user_events: %v", err)
	}
	fmt.Println("  ✓ Subscribed to 'user_events' channel")

	// System notifications subscription
	systemSub, err := sc.Subscribe(ctx, "system", func(ctx context.Context, msg *cable.Message) error {
		fmt.Printf("  🔔 [system] Alert: %v\n", msg.Data)
		return nil
	})
	if err != nil {
		log.Printf("Failed to subscribe to system: %v", err)
	}
	fmt.Println("  ✓ Subscribed to 'system' channel")

	// Global subscription (receives all messages)
	globalSub, err := sc.Subscribe(ctx, "*", func(ctx context.Context, msg *cable.Message) error {
		fmt.Printf("  🌍 [global] Channel '%s': %v\n", msg.Channel, msg.Data)
		return nil
	})
	if err != nil {
		log.Printf("Failed to subscribe to *: %v", err)
	}
	fmt.Println("  ✓ Subscribed to all channels (*)")

	// Give subscriptions time to setup
	time.Sleep(100 * time.Millisecond)

	// Publish messages
	fmt.Println("\n  Publishing messages...")

	// User event
	if err := sc.Publish(ctx, "user_events", map[string]interface{}{
		"event": "user_registered",
		"user_id": 123,
		"email": "newuser@example.com",
	}); err != nil {
		log.Printf("Failed to publish user event: %v", err)
	} else {
		fmt.Println("  ✓ Published user registration event")
	}

	// System notification
	if err := sc.PublishWithMetadata(ctx, "system", 
		map[string]interface{}{
			"level": "warning",
			"message": "High memory usage detected",
		},
		map[string]string{
			"severity": "medium",
			"component": "cache",
		},
	); err != nil {
		log.Printf("Failed to publish system notification: %v", err)
	} else {
		fmt.Println("  ✓ Published system warning")
	}

	// Another user event
	if err := sc.Publish(ctx, "user_events", map[string]interface{}{
		"event": "user_login",
		"user_id": 123,
		"ip": "192.168.1.1",
	}); err != nil {
		log.Printf("Failed to publish login event: %v", err)
	} else {
		fmt.Println("  ✓ Published user login event")
	}

	// Broadcast to all
	if err := sc.Broadcast(ctx, map[string]interface{}{
		"announcement": "System maintenance in 1 hour",
		"timestamp": time.Now().Format(time.RFC3339),
	}); err != nil {
		log.Printf("Failed to broadcast: %v", err)
	} else {
		fmt.Println("  ✓ Broadcast system announcement")
	}

	// Wait for messages to be processed
	fmt.Println("\n  Processing messages...")
	time.Sleep(1 * time.Second)

	// Get stats
	stats, err := sc.GetStats()
	if err == nil {
		fmt.Println("\n  Cable Statistics:")
		fmt.Printf("    Total subscriptions: %v\n", stats["total_subscriptions"])
		fmt.Printf("    Total channels: %v\n", stats["total_channels"])
		fmt.Printf("    Total messages: %v\n", stats["total_messages"])
		if channels, ok := stats["channels"].(map[string]int); ok {
			fmt.Println("    Subscriptions per channel:")
			for channel, count := range channels {
				fmt.Printf("      %s: %d\n", channel, count)
			}
		}
	}

	// List active channels
	channels := sc.ListChannels()
	fmt.Printf("\n  Active channels: %v\n", channels)

	// Cleanup
	fmt.Println("\n  Cleaning up subscriptions...")
	sc.Unsubscribe(userSub)
	fmt.Println("  ✓ Unsubscribed from 'user_events'")
	sc.Unsubscribe(systemSub)
	fmt.Println("  ✓ Unsubscribed from 'system'")
	sc.Unsubscribe(globalSub)
	fmt.Println("  ✓ Unsubscribed from global")

	fmt.Println("")
}