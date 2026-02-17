package services

import (
	"github.com/jmoiron/sqlx"
	"github.com/school-invoice/backend/internal/config"
	"github.com/school-invoice/backend/internal/database"
	"github.com/school-invoice/backend/internal/logger"
)

// EducationService contains all dependencies for the school invoice service
type EducationService struct {
	DBX    *sqlx.DB
	Redis  *database.Redis
	Config *config.Config
	Log    *logger.Logger
}

// NewEducationService creates a new EducationService instance
func NewEducationService(db *sqlx.DB, redis *database.Redis, cfg *config.Config, log *logger.Logger) *EducationService {
	return &EducationService{
		DBX:    db,
		Redis:  redis,
		Config: cfg,
		Log:    log,
	}
}
