package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/logger"
)

func ValidateCreateTermRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.CreateTermRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.SchoolID = GetSchoolID(c)
	req.UserID = GetUserID(c)
	if !req.Name.IsValid() {
		logger.
			WithField("user_id", req.UserID).
			WithField("school_id", req.SchoolID).
			WithField("session_id", req.SessionID).
			Error("Invalid term name")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid term name"})
		return
	}
	if req.SortOrder < 1 {
		logger.
			WithField("user_id", req.UserID).
			WithField("school_id", req.SchoolID).
			WithField("session_id", req.SessionID).
			Error("Sort order must be greater than 0")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Sort order must be greater than 0"})
		return
	}

	if req.StartDate != "" {
		var err error
		req.Start, err = time.Parse(DateFormat, req.StartDate)
		if err != nil {
			logger.
				WithError(err).
				WithField("user_id", req.UserID).
				WithField("school_id", req.SchoolID).
				WithField("session_id", req.SessionID).
				Error("Invalid start date")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid start date"})
			return
		}
	}
	if req.EndDate != "" {
		var err error
		req.End, err = time.Parse(DateFormat, req.EndDate)
		if err != nil {
			logger.
				WithError(err).
				WithField("user_id", req.UserID).
				WithField("school_id", req.SchoolID).
				WithField("session_id", req.SessionID).
				Error("Invalid end date")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid end date"})
			return
		}
	}
	if req.Start.After(req.End) {
		logger.
			WithField("user_id", req.UserID).
			WithField("school_id", req.SchoolID).
			WithField("session_id", req.SessionID).
			WithField("start_date", req.Start).
			WithField("end_date", req.End).
			Error("Start date must be before end date")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Start date must be before end date"})
		return
	}
	c.Set(ReqBodyCreateTerm, req)
	c.Next()

}

func ValidateUpdateTermRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateTermRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.SchoolID = GetSchoolID(c)
	req.UserID = GetUserID(c)
	termID := c.Param("id")
	if termID == "" {
		logger.Error("Term ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Term ID is required"})
		return
	}
	req.TermID = uuid.MustParse(termID)
	if req.Name != "" {
		if !req.Name.IsValid() {
			logger.
				WithField("user_id", req.UserID).
				WithField("school_id", req.SchoolID).
				WithField("session_id", req.SessionID).
				Error("Invalid term name")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid term name"})
			return
		}
		if req.StartDate != "" {
			var err error
			req.Start, err = time.Parse(DateFormat, req.StartDate)
			if err != nil {
				logger.
					WithError(err).
					WithField("user_id", req.UserID).
					WithField("school_id", req.SchoolID).
					WithField("session_id", req.SessionID).
					Error("Invalid start date")
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid start date"})
				return
			}
		}
		if req.EndDate != "" {
			var err error
			req.End, err = time.Parse(DateFormat, req.EndDate)
			if err != nil {
				logger.
					WithError(err).
					WithField("user_id", req.UserID).
					WithField("school_id", req.SchoolID).
					WithField("session_id", req.SessionID).
					Error("Invalid end date")
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid end date"})
				return
			}
		}
		if req.Start.After(req.End) {
			logger.
				WithField("user_id", req.UserID).
				WithField("school_id", req.SchoolID).
				WithField("session_id", req.SessionID).
				WithField("start_date", req.Start).
				WithField("end_date", req.End).
				Error("Start date must be before end date")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Start date must be before end date"})
			return
		}
		c.Set(ReqBodyUpdateTerm, req)
		c.Next()
	}
}

func ValidateSetCurrentTermRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.SetCurrentTermRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.SchoolID = GetSchoolID(c)
	req.UserID = GetUserID(c)
	termID := c.Param("id")
	if termID == "" {
		logger.
			WithField("user_id", req.UserID).
			WithField("school_id", req.SchoolID).
			Error("Term ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Term ID is required"})
		return
	}
	req.TermID = uuid.MustParse(termID)

	c.Set(ReqBodySetCurrentTerm, req)
	c.Next()
}
