package models

import (
	"github.com/google/uuid"
)

// Register is the request body for school registration
type Register struct {
	// School details
	SchoolName    string `json:"school_name" binding:"required,min=2,max=255"`
	Subdomain     string `json:"subdomain" binding:"required,min=3,max=50,alphanum"`
	SchoolPhone   string `json:"school_phone" binding:"required"`
	SchoolEmail   string `json:"school_email" binding:"required,email"`
	SchoolAddress string `json:"school_address" binding:"required"`

	// Super Admin details
	AdminEmail     string `json:"admin_email" binding:"required,email"`
	AdminPassword  string `json:"admin_password" binding:"required,min=8"`
	AdminFirstName string `json:"admin_first_name" binding:"required,min=2"`
	AdminLastName  string `json:"admin_last_name" binding:"required,min=2"`
	AdminPhone     string `json:"admin_phone" binding:"required"`
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

// ResetPasswordRequest is the request body for resetting password
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
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

func (r *Register) DomainExists(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schools WHERE subdomain = $1)", r.Subdomain).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
