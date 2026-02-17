package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/school-invoice/backend/internal/config"
	"github.com/school-invoice/backend/internal/database"
	"github.com/school-invoice/backend/internal/logger"
)

var log *logger.Logger

func main() {
	// Initialize logger
	log = logger.New("school-invoice-worker")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set log level based on environment
	if cfg.IsProduction() {
		log.SetLevel("info")
	} else {
		log.SetLevel("debug")
	}

	// Connect to database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Info("Worker connected to PostgreSQL database")

	// Connect to Redis
	redis, err := database.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()
	log.Info("Worker connected to Redis")

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker goroutines
	go processReminderQueue(ctx, db, redis, cfg)
	go processOverdueInvoices(ctx, db, cfg)

	log.Info("Worker started, processing background jobs...")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down worker...")

	cancel()
	time.Sleep(2 * time.Second) // Give goroutines time to finish

	log.Info("Worker exited gracefully")
}

// processReminderQueue processes SMS reminder jobs from the queue
func processReminderQueue(ctx context.Context, db *database.DB, redis *database.Redis, cfg *config.Config) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Try to get a job from the reminder queue
			data, err := redis.Dequeue(ctx, "reminders", 5*time.Second)
			if err != nil {
				// Timeout or error, continue
				continue
			}

			// Process the reminder
			log.Infof("Processing reminder: %s", data)
			// TODO: Implement actual SMS sending via Termii
		}
	}
}

// processOverdueInvoices checks for overdue invoices and updates their status
func processOverdueInvoices(ctx context.Context, db *database.DB, cfg *config.Config) {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Info("Checking for overdue invoices...")

			// Update invoices that are past due date
			_, err := db.ExecContext(ctx, `
				UPDATE invoices 
				SET status = 'overdue', updated_at = NOW()
				WHERE status IN ('pending', 'partial')
				AND due_date < CURRENT_DATE
				AND (grace_date IS NULL OR grace_date < CURRENT_DATE)
			`)
			if err != nil {
				log.WithError(err).Error("Error updating overdue invoices")
			} else {
				log.Info("Overdue invoices check completed")
			}
		}
	}
}
