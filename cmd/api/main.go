package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/school-invoice/backend/internal/config"
	"github.com/school-invoice/backend/internal/database"
	"github.com/school-invoice/backend/internal/logger"
	"github.com/school-invoice/backend/internal/services"
	"github.com/school-invoice/backend/internal/middleware"
	"github.com/gin-gonic/gin/binding"
    "github.com/go-playground/validator/v10"
)

func main() {
	// Initialize logger
	log := logger.New("school-invoice-api")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set log level based on environment
	if cfg.IsProduction() {
		log.SetLevel("info")
		gin.SetMode(gin.ReleaseMode)
	} else {
		log.SetLevel("debug")
	}

	// Connect to database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Info("Connected to PostgreSQL database")

	// Connect to Redis
	redis, err := database.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()
	log.Info("Connected to Redis")

	// Initialize EducationService with all dependencies
	educationService := services.NewEducationService(db.DB, redis, cfg, log)

	// Setup router with all routes
	router := educationService.SetupRouter()
	// Register custom "phone" tag
    if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
        v.RegisterValidation("phone", middleware.PhoneValidator)
    }

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Infof("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited gracefully")
}
