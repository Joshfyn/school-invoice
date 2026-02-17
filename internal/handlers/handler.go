package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/school-invoice/backend/internal/config"
	"github.com/school-invoice/backend/internal/database"
	"github.com/school-invoice/backend/internal/logger"
	"github.com/school-invoice/backend/internal/models"
)

// Handler contains all HTTP handlers
type Handler struct {
	dbx    *sqlx.DB
	redis  *database.Redis
	config *config.Config
	logger *logger.Logger
}

// New creates a new Handler instance
func New(dbx *sqlx.DB, redis *database.Redis, cfg *config.Config, log *logger.Logger) *Handler {
	return &Handler{
		dbx:    dbx,
		redis:  redis,
		config: cfg,
		logger: log,
	}
}

// Health returns the health status of the service
func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	dbStatus := "healthy"
	if err := h.dbx.PingContext(ctx); err != nil {
		dbStatus = "unhealthy"
	}

	redisStatus := "healthy"
	if err := h.redis.Health(ctx); err != nil {
		redisStatus = "unhealthy"
	}

	status := "healthy"
	if dbStatus == "unhealthy" || redisStatus == "unhealthy" {
		status = "degraded"
	}

	c.JSON(http.StatusOK, models.HealthResponse{
		Status:   status,
		Database: dbStatus,
		Redis:    redisStatus,
		Version:  "1.0.0",
	})
}

// respondWithError sends an error response
func respondWithError(c *gin.Context, status int, err, message string) {
	c.JSON(status, models.ErrorResponse{
		Error:   err,
		Message: message,
	})
}
