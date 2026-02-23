package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/logger"
)

func ValidateResetToken(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token := c.Query("token")
	if token == "" {
		logger.Error("Unauthorized: Token is required")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	req.Token = token

	c.Set(ReqBodyResetPwd, req)
	c.Next()
}

func ValidateCreateUserRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Set(ReqBodyCreateUser, req)
	c.Next()
}

func ValidateGetSingleUserRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.GetSingleUserRequest
	userID := c.Param("id")
	if userID == "" {
		logger.Error("User ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}
	isSuperAdmin := GetIsSuperAdmin(c)
	authUserID := GetUserID(c)
	schoolID := GetSchoolID(c)

	if !isSuperAdmin && authUserID != uuid.MustParse(userID) {
		logger.Error("Unauthorized: User ID does not match authenticated user ID")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	req.SchoolID = schoolID
	req.IsSuperAdmin = isSuperAdmin
	req.UserID = uuid.MustParse(userID)
	c.Set(ReqBodyGetSingleUser, req)
	c.Next()
}

func ValidateUpdateUserRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.Param("id")
	if userID == "" {
		logger.Error("User ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}
	req.UserID = uuid.MustParse(userID)
	c.Set(ReqBodyUpdateUser, req)
	c.Next()
}

func ValidateUpdateUserRoleRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.Param("id")
	if userID == "" {
		logger.Error("User ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}
	req.UserID = uuid.MustParse(userID)

	c.Set(ReqBodyUpdateUserRole, req)
	c.Next()
}

func ValidateUpdateUserStatusRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.Param("id")
	if userID == "" {
		logger.Error("User ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}
	req.UserID = uuid.MustParse(userID)
	c.Set(ReqBodyUpdateUserStatus, req)
	c.Next()
}
