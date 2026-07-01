package middleware

import (
	"context"
	"database/sql"
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
	ContextKeyUserID       = "user_id"
	ContextKeySchoolID     = "school_id"
	ContextKeyRoleID       = "role_id"
	ContextKeyIsSuperAdmin = "is_super_admin"
	ContextKeyRole         = "role"
	ContextKeyUser         = "user"
)

// PermissionCheck pairs a resource with an action for RequireAnyPermission.
type PermissionCheck struct {
	Resource string
	Action   string
}

// Auth middleware validates JWT token, loads the user's role, and sets context.
func Auth(jwtSecret string, redis *database.Redis, dbx models.DBTX) gin.HandlerFunc {
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
		tokenUserID, err := redis.Get(context.Background(), tokenString)
		if err != nil || tokenUserID == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

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

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid token claims",
			})
			c.Abort()
			return
		}

		userID, _ := uuid.Parse(claims["user_id"].(string))
		if tokenUserID != userID.String() {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid token",
			})
			c.Abort()
			return
		}

		schoolIDStr, ok := claims["school_id"].(string)
		roleIDStr, okRole := claims["role_id"].(string)
		if !ok || !okRole {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid token claims",
			})
			c.Abort()
			return
		}
		schoolID, err := uuid.Parse(schoolIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized", Message: "Invalid token claims"})
			c.Abort()
			return
		}
		roleID, err := uuid.Parse(roleIDStr)
		if err != nil || roleID == uuid.Nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid role in token — please log in again",
			})
			c.Abort()
			return
		}
		isSuperAdmin, _ := claims["is_super_admin"].(bool)

		role, err := models.GetRole(dbx, schoolID, roleID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusForbidden, models.ErrorResponse{
					Error:   "forbidden",
					Message: "Role not found or access denied",
				})
				c.Abort()
				return
			}
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "server_error",
				Message: "Failed to load user role",
			})
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeySchoolID, schoolID)
		c.Set(ContextKeyRoleID, roleID)
		c.Set(ContextKeyIsSuperAdmin, isSuperAdmin || role.IsSuperAdmin)
		SetRole(c, role)
		c.Next()
	}
}

func GetUserID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(ContextKeyUserID); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

func GetSchoolID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(ContextKeySchoolID); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

func GetRoleID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(ContextKeyRoleID); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

func GetIsSuperAdmin(c *gin.Context) bool {
	if isSuperAdmin, exists := c.Get(ContextKeyIsSuperAdmin); exists {
		return isSuperAdmin.(bool)
	}
	return false
}

func GetRole(c *gin.Context) *models.Role {
	if role, exists := c.Get(ContextKeyRole); exists {
		return role.(*models.Role)
	}
	return nil
}

func SetRole(c *gin.Context, role *models.Role) {
	c.Set(ContextKeyRole, role)
}

func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetRole(c)
		if role == nil {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to perform this action",
			})
			c.Abort()
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

func RequireAnyPermission(checks ...PermissionCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetRole(c)
		if role == nil {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to perform this action",
			})
			c.Abort()
			return
		}

		for _, check := range checks {
			if role.HasPermission(check.Resource, check.Action) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "You don't have permission to perform this action",
		})
		c.Abort()
	}
}
