package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/logger"
)

func ValidateCreateStudentRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Set(ReqBodyCreateStudent, req)
	c.Next()
}

func ValidateUpdateStudentRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	studentID := c.Param("id")
	if studentID == "" {
		logger.Error("Student ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Student ID is required"})
		return
	}
	req.StudentID = uuid.MustParse(studentID)
	if !req.DateOfBirth.IsZero() {
		logger.Error("Date of birth is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Date of birth is required"})
		return
	}

	c.Set(ReqBodyUpdateStudent, req)
	c.Next()
}

func ValidateGetSingleStudentRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.GetSingleStudentRequest
	studentID := c.Param("id")
	if studentID == "" {
		logger.Error("Student ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Student ID is required"})
		return
	}
	req.StudentID = uuid.MustParse(studentID)
	c.Set(ReqBodyGetSingleStudent, req)
	c.Next()
}
