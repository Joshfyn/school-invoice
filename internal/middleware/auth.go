package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/database"
	"github.com/school-invoice/backend/internal/models"
)

// Context keys for storing user info
const (
	ContextKeyUserID   = "user_id"
	ContextKeySchoolID = "school_id"
	ContextKeyRoleID   = "role_id"
	ContextKeyRole     = "role"
	ContextKeyUser     = "user"
)

// Auth middleware validates JWT token and sets user context
func Auth(jwtSecret string, redis *database.Redis) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		// check if token is in redis
		tokenUserID, err := redis.Get(context.Background(), tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		if tokenUserID == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid token claims",
			})
			c.Abort()
			return
		}

		// Set user info in context
		userID, _ := uuid.Parse(claims["user_id"].(string))
		if tokenUserID != userID.String() {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid token",
			})
			c.Abort()
			return
		}

		schoolID, _ := uuid.Parse(claims["school_id"].(string))
		roleID, _ := uuid.Parse(claims["role_id"].(string))

		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeySchoolID, schoolID)
		c.Set(ContextKeyRoleID, roleID)

		c.Next()
	}
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(ContextKeyUserID); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

// GetSchoolID retrieves school ID from context
func GetSchoolID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(ContextKeySchoolID); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

// GetRoleID retrieves role ID from context
func GetRoleID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(ContextKeyRoleID); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

// GetRole retrieves role from context
func GetRole(c *gin.Context) *models.Role {
	if role, exists := c.Get(ContextKeyRole); exists {
		return role.(*models.Role)
	}
	return nil
}

// SetRole sets role in context (called after loading role from DB)
func SetRole(c *gin.Context, role *models.Role) {
	c.Set(ContextKeyRole, role)
}

// RequirePermission middleware checks if user has required permission
func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetRole(c)
		if role == nil {
			// Role not loaded yet, permission check will be done in handler
			// For now, allow pass-through (handler should load role and check)
			c.Next()
			return
		}

		if !role.HasPermission(resource, action) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to perform this action",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
