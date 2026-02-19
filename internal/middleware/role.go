package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/logger"
	"github.com/school-invoice/backend/internal/models"
)

func ValidateCreateRoleRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		logger.
			Error("Name is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}

	permissions := models.Permissions(req.Permissions)
	if err := permissions.ValidateStrict(models.PermissionsValidationOptions{AllowWildcard: false, AllowEmpty: false}); err != nil {
		logger.
			WithError(err).
			Error("Failed to validate permissions")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Set(ReqBodyCreateRole, req)
	c.Next()
}

func ValidateUpdateRoleRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		logger.
			WithError(err).
			Error("Failed to parse role ID")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.RoleID = roleID

	if req.Permissions != nil {
		permissions := models.Permissions(*req.Permissions)
		if err := permissions.ValidateStrict(models.PermissionsValidationOptions{AllowWildcard: false, AllowEmpty: false}); err != nil {
			logger.
				WithError(err).
				Error("Failed to validate permissions")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		req.Permissions = &permissions
	}
	c.Set(ReqBodyUpdateRole, req)
	c.Next()
}

func ValidateDeleteRoleRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		logger.
			WithError(err).
			Error("Failed to parse role ID")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if roleID == uuid.Nil {
		logger.
			Error("Role ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Role ID is required"})
		return
	}

	var req = dto.DeleteRoleRequest{
		RoleID: roleID,
	}
	c.Set(ReqBodyDeleteRole, req)
	c.Next()
}
