package dto

import (
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/models"
)

// CreateUserRequest is the request body for creating a user
type CreateUserRequest struct {
	RoleID    uuid.UUID `json:"role_id" binding:"required"`
	Email     string    `json:"email" binding:"required,email"`
	Password  string    `json:"password" binding:"required,min=8"`
	FirstName string    `json:"first_name" binding:"required,min=2"`
	LastName  string    `json:"last_name" binding:"required,min=2"`
	Phone     string    `json:"phone" binding:"required"`
}

// GetSingleUserRequest is the request body for getting a single user
type GetSingleUserRequest struct {
	UserID       uuid.UUID `json:"-"`
	SchoolID     uuid.UUID `json:"-"`
	IsSuperAdmin bool      `json:"-"`
}

// UpdateUserRequest is the request body for updating a user
type UpdateUserRequest struct {
	UserID    uuid.UUID `json:"-"`
	FirstName string    `json:"first_name,omitempty" binding:"omitempty,min=2"`
	LastName  string    `json:"last_name,omitempty" binding:"omitempty,min=2"`
	Phone     string    `json:"phone,omitempty"`
}

// UpdateUserRoleRequest is the request body for changing a user's role
type UpdateUserRoleRequest struct {
	UserID uuid.UUID `json:"-"`
	RoleID uuid.UUID `json:"role_id" binding:"required"`
}

// UpdateUserStatusRequest is the request body for activating/deactivating a user
type UpdateUserStatusRequest struct {
	UserID   uuid.UUID `json:"-"`
	IsActive *bool     `json:"is_active"`
}

// UserResponse is the response for user data
type UserResponse struct {
	ID        uuid.UUID            `json:"id"`
	SchoolID  uuid.UUID            `json:"school_id"`
	RoleID    uuid.UUID            `json:"role_id"`
	Email     string               `json:"email"`
	FirstName string               `json:"first_name"`
	LastName  string               `json:"last_name"`
	Phone     string               `json:"phone"`
	IsActive  bool                 `json:"is_active"`
	Role      *models.RoleResponse `json:"role,omitempty"`
}

// RegisterResponse is the response for school registration
type RegisterResponse struct {
	School SchoolResponse `json:"school"`
	User   UserResponse   `json:"user"`
	Token  string         `json:"token"`
}

// LoginRequest is the request body for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the response for user login
type LoginResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

// ForgotPasswordRequest is the request body for password reset request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ChangePasswordRequest is the request body for changing password (logged in user)
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// TokenClaims represents the JWT token claims
type TokenClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	SchoolID uuid.UUID `json:"school_id"`
	RoleID   uuid.UUID `json:"role_id"`
	Email    string    `json:"email"`
}

// RefreshTokenRequest is the request body for refreshing access token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Token       string `json:"-"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
