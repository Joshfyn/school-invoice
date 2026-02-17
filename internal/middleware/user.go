package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/logger"
)

func ValidateResetToken(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to bind JSON")
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
