package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/logger"
)

func ValidateCreateGuardianRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.CreateGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Validate NIN format against Nigerian NIN format and against Nigerian agency

	c.Set(ReqBodyCreateGuardian, req)
	c.Next()
}

func ValidateUpdateGuardianRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	guardianID := c.Param("id")
	if guardianID == "" {
		logger.Error("Guardian ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Guardian ID is required"})
		return
	}
	req.GuardianID = uuid.MustParse(guardianID)
	c.Set(ReqBodyUpdateGuardian, req)
	c.Next()
}
