package handlers

import (
	"github.com/school-invoice/backend/internal/config"
	"github.com/school-invoice/backend/internal/database"
	"github.com/school-invoice/backend/internal/logger"
)

type InvoiceHandler struct {
	DB     *database.DB
	Redis  *database.Redis
	Config *config.Config
	Logger *logger.Logger
}

