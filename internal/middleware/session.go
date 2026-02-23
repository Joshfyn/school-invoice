package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/logger"
)

const (
	DateFormat = "2006-01-02" // YYYY-MM-DD
)

func ValidateCreateSessionRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	schoolID := GetSchoolID(c)
	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.StartDate == "" {
		logger.Error("Start date is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Start date is required"})
		return
	}
	if req.EndDate == "" {
		logger.Error("End date is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "End date is required"})
		return
	}

	startDate, err := time.Parse(DateFormat, req.StartDate)
	if err != nil {
		logger.
			WithError(err).
			Error("Invalid start date")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid start date"})
		return
	}
	endDate, err := time.Parse(DateFormat, req.EndDate)
	if err != nil {
		logger.
			WithError(err).
			Error("Invalid end date")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid end date"})
		return
	}
	if startDate.After(endDate) {
		logger.
			WithError(errors.New("start date must be before end date")).
			Error("Start date must be before end date")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Start date must be before end date"})
		return
	}
	req.Name = fmt.Sprintf("%d/%d", startDate.Year(), endDate.Year())
	req.Start = startDate
	req.End = endDate
	req.SchoolID = schoolID

	c.Set(ReqBodyCreateSession, req)
	c.Next()
}

func ValidateUpdateSessionRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sessionID := c.Param("id")
	if sessionID == "" {
		logger.Error("Session ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}
	req.SessionID = uuid.MustParse(sessionID)

	if req.StartDate != "" {
		var err error
		req.Start, err = time.Parse(DateFormat, req.StartDate)
		if err != nil {
			logger.WithError(err).Error("Invalid start date")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid start date"})
			return
		}
	}
	if req.EndDate != "" {
		var err error
		req.End, err = time.Parse(DateFormat, req.EndDate)
		if err != nil {
			logger.WithError(err).Error("Invalid end date")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid end date"})
			return
		}
	}

	if req.StartDate != "" && req.EndDate != "" {
		if req.Start.After(req.End) {
			logger.Error("Start date must be before end date")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Start date must be before end date"})
			return
		}
		req.Name = fmt.Sprintf("%d/%d", req.Start.Year(), req.End.Year())
	}

	req.SchoolID = GetSchoolID(c)
	c.Set(ReqBodyUpdateSession, req)
	c.Next()
}

func ValidateSetCurrentSessionRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.SetCurrentSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sessionID := c.Param("id")
	if sessionID == "" {
		logger.Error("Session ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}
	req.SessionID = uuid.MustParse(sessionID)
	req.SchoolID = GetSchoolID(c)
	c.Set(ReqBodySetCurrentSession, req)
	c.Next()
}