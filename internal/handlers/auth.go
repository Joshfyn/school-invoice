package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/middleware"
	"github.com/school-invoice/backend/internal/models"
	"github.com/school-invoice/backend/lib/mail"
	"golang.org/x/crypto/bcrypt"
)

const (
	ResetTokenExpiry = time.Hour * 1
)

// Register handles school registration
func (h *Handler) Register(c *gin.Context) {
	var regReq models.Register
	if err := c.ShouldBindJSON(&regReq); err != nil {
		h.logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Check if subdomain is already taken
	exists, err := regReq.DomainExists(h.dbx)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("subdomain", regReq.Subdomain).
			Error("Failed to check subdomain")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to check subdomain",
		})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:   "subdomain_taken",
			Message: "This subdomain is already registered",
		})
		return
	}

	// Check if admin email is already used
	exists, err = (&models.User{Email: regReq.AdminEmail}).EmailExists(h.dbx)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("email", regReq.AdminEmail).
			Error("Failed to check email")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to check email",
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:   "email_taken",
			Message: "This email is already registered",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(regReq.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("email", regReq.AdminEmail).
			Error("Failed to process password")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to process password",
		})
		return
	}

	// Start transaction
	tx := h.dbx.MustBegin()
	defer tx.Rollback()

	// Create school
	schoolID := uuid.New()
	now := time.Now().UTC()
	err = (&models.School{
		BaseModel: models.BaseModel{
			ID:        schoolID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name:      regReq.SchoolName,
		Subdomain: regReq.Subdomain,
		Phone:     regReq.SchoolPhone,
		Email:     regReq.SchoolEmail,
		Address:   regReq.SchoolAddress,
		IsActive:  true,
	}).Create(tx)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			Error("Failed to create school")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to create school",
		})
		return
	}

	if err := models.SeedFeeCategories(tx, schoolID); err != nil {
		h.logger.WithError(err).WithField("school_id", schoolID).Error("Failed to seed fee categories")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to initialize school settings",
		})
		return
	}

	// Create super admin role
	roleID := uuid.New()
	permissions := models.GetSuperAdminPermissions()
	err = (&models.Role{
		BaseModel: models.BaseModel{
			ID:        roleID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		SchoolID:     schoolID,
		Name:         "Super Admin",
		Description:  "Full access to all features",
		Permissions:  permissions,
		IsSuperAdmin: true,
	}).Create(tx)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("role_id", roleID).
			Error("Failed to create role")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to create role",
		})
		return
	}

	// Create admin user
	userID := uuid.New()
	err = (&models.User{
		BaseModel: models.BaseModel{
			ID:        userID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		SchoolID:     schoolID,
		RoleID:       roleID,
		Email:        regReq.AdminEmail,
		PasswordHash: string(hashedPassword),
		FirstName:    regReq.AdminFirstName,
		LastName:     regReq.AdminLastName,
		Phone:        regReq.AdminPhone,
		IsActive:     func(v bool) *bool { return &v }(true),
	}).Create(tx)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("user_id", userID).
			Error("Failed to create user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to create user",
		})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		h.logger.
			WithField("email", regReq.AdminEmail).
			WithField("school_id", schoolID).
			WithField("role_id", roleID).
			WithField("user_id", userID).
			WithError(err).
			Error("Failed to commit transaction")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to complete registration",
		})
		return
	}

	// Generate JWT token
	roleResp := (&models.Role{
		BaseModel:    models.BaseModel{ID: roleID},
		SchoolID:     schoolID,
		Name:         "Super Admin",
		IsSuperAdmin: true,
		Permissions:  models.GetSuperAdminPermissions(),
	}).ToResponse()

	token, err := h.generateToken(TokenClaimsRequirement{
		UserID:       userID,
		SchoolID:     schoolID,
		RoleID:       roleID,
		Email:        regReq.AdminEmail,
		IsSuperAdmin: true,
	})
	if err != nil {
		h.logger.
			WithError(err).
			WithField("user_id", userID).
			WithField("school_id", schoolID).
			WithField("role_id", roleID).
			WithField("email", regReq.AdminEmail).
			Error("Failed to generate token")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to generate token",
		})
		return
	}

	// TODO: send welcom and confiirmation request email to admin

	c.JSON(http.StatusCreated, dto.RegisterResponse{
		School: dto.SchoolResponse{
			ID:        schoolID,
			Name:      regReq.SchoolName,
			Subdomain: regReq.Subdomain,
			Phone:     regReq.SchoolPhone,
			Email:     regReq.SchoolEmail,
			Address:   regReq.SchoolAddress,
			IsActive:  true,
		},
		User: dto.UserResponse{
			ID:        userID,
			SchoolID:  schoolID,
			RoleID:    roleID,
			Email:     regReq.AdminEmail,
			FirstName: regReq.AdminFirstName,
			LastName:  regReq.AdminLastName,
			Phone:     regReq.AdminPhone,
			IsActive:  true,
			Role:      &roleResp,
		},
		Token: token,
	})
}

// GetMe returns the currently authenticated user and role.
func (h *Handler) GetMe(c *gin.Context) {
	uid := middleware.GetUserID(c)
	schoolID := middleware.GetSchoolID(c)
	role := middleware.GetRole(c)

	userWithRole, err := (&models.User{BaseModel: models.BaseModel{ID: uid}}).GetUser(h.dbx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "user_not_found", Message: "User not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get current user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "server_error", Message: "Failed to get user"})
		return
	}

	if userWithRole.User.SchoolID != schoolID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "forbidden", Message: "Access denied"})
		return
	}

	var roleResp *models.RoleResponse
	if role != nil {
		r := role.ToResponse()
		roleResp = &r
	}

	isActive := userWithRole.User.IsActive != nil && *userWithRole.User.IsActive
	c.JSON(http.StatusOK, dto.UserResponse{
		ID:        userWithRole.User.ID,
		SchoolID:  userWithRole.User.SchoolID,
		RoleID:    userWithRole.User.RoleID,
		Email:     userWithRole.User.Email,
		FirstName: userWithRole.User.FirstName,
		LastName:  userWithRole.User.LastName,
		Phone:     userWithRole.User.Phone,
		IsActive:  isActive,
		Role:      roleResp,
	})
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Find user by email
	userAndRole, err := (&models.User{Email: req.Email}).FindByEmail(h.dbx)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.
				WithField("email", req.Email).
				Error("User not found")
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_credentials",
				Message: "User not found",
			})
			return
		}
		h.logger.
			WithError(err).
			WithField("email", req.Email).
			Error("Failed to find user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to find user",
		})
		return
	}

	// Check if user is active
	if userAndRole.User.IsActive == nil || !*userAndRole.User.IsActive {
		h.logger.
			WithField("email", req.Email).
			WithField("user_id", userAndRole.User.ID).
			Error("User is disabled")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "account_disabled",
			Message: "Your account has been disabled",
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(userAndRole.User.PasswordHash), []byte(req.Password)); err != nil {
		respondWithError(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	roleID := userAndRole.User.RoleID
	if userAndRole.Role.ID != uuid.Nil {
		roleID = userAndRole.Role.ID
	}

	// Generate JWT token
	token, err := h.generateToken(TokenClaimsRequirement{
		UserID:       userAndRole.User.ID,
		SchoolID:     userAndRole.User.SchoolID,
		RoleID:       roleID,
		Email:        userAndRole.User.Email,
		IsSuperAdmin: userAndRole.Role.IsSuperAdmin,
	})
	if err != nil {
		h.logger.
			WithError(err).
			WithField("user_id", userAndRole.User.ID).
			WithField("school_id", userAndRole.User.SchoolID).
			WithField("role_id", userAndRole.Role.ID).
			WithField("email", userAndRole.User.Email).
			Error("Failed to generate token")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to generate token",
		})
		return
	}
	// set token to redis
	err = h.redis.Set(context.Background(), token, userAndRole.User.ID.String(), ResetTokenExpiry)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("token", token).
			Error("Failed to set token to redis")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to set token to redis",
		})
		return
	}

	roleResp := userAndRole.Role.ToResponse()
	c.JSON(http.StatusOK, dto.LoginResponse{
		User: dto.UserResponse{
			ID:        userAndRole.User.ID,
			SchoolID:  userAndRole.User.SchoolID,
			RoleID:    userAndRole.Role.ID,
			Email:     userAndRole.User.Email,
			FirstName: userAndRole.User.FirstName,
			LastName:  userAndRole.User.LastName,
			Phone:     userAndRole.User.Phone,
			IsActive:  userAndRole.User.IsActive == nil || *userAndRole.User.IsActive,
			Role:      &roleResp,
		},
		Token: token,
	})
}

// ForgotPassword handles password reset request
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	var user = models.User{Email: req.Email}
	userAndRole, err := user.FindByEmail(h.dbx)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.
				WithField("email", req.Email).
				Error("User not found")
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "user_not_found",
				Message: "No account found with this email",
			})
			return
		}
		h.logger.
			WithError(err).
			WithField("email", req.Email).
			Error("Failed to find user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to find user",
		})
		return
	}
	user = userAndRole.User

	// 1. Generate JWT token
	resetToken, err := h.generateToken(TokenClaimsRequirement{
		UserID:   user.ID,
		SchoolID: user.SchoolID,
		RoleID:   user.RoleID,
		Email:    user.Email,
		Exp:      time.Now().Add(ResetTokenExpiry).Unix(),
		Iat:      time.Now().Unix(),
	})
	if err != nil {
		h.logger.
			WithError(err).
			WithField("user_id", user.ID).
			WithField("school_id", user.SchoolID).
			WithField("role_id", user.RoleID).
			WithField("email", req.Email).
			Error("Failed to generate token")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to generate token",
		})
		return
	}

	// 2. Store token with expiry
	err = h.redis.Set(context.Background(), resetToken, user.ID.String(), ResetTokenExpiry)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("reset_token", resetToken).
			Error("Failed to store token")
	}
	if err != nil {
		h.logger.
			WithError(err).
			WithField("reset_token", resetToken).
			Error("Failed to store token")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to store token",
		})
		return
	}

	// 3. Send email/SMS with reset link
	err = mail.SendResetPasswordEmail(user.Email, resetToken)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("email", user.Email).
			Error("Failed to send reset password email")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to send reset password email",
		})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Password reset link has been sent to your email",
	})
}

// ResetPassword handles password reset
func (h *Handler) ResetPassword(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyResetPwd).(dto.ResetPasswordRequest)

	// 1. Validate reset token
	userID, err := h.redis.Get(context.Background(), req.Token)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("reset_token", req.Token).
			Error("Failed to validate reset token")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid or expired reset token",
		})
		return
	}
	if userID == "" {
		h.logger.
			WithField("reset_token", req.Token).
			Error("Reset token not found")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid or expired reset token",
		})
		return
	}

	// get email from token claims
	tokenDetails, err := h.GetTokenDetails(req.Token)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("reset_token", req.Token).
			Error("Failed to get token details")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to get token details",
		})
		return
	}

	// check if email matches the user email
	if tokenDetails.UserID.String() != userID {
		h.logger.
			WithField("reset_token", req.Token).
			WithField("user_id", tokenDetails.UserID).
			WithField("user_id", userID).
			Error("User ID does not match")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID does not match",
		})
		return
	}

	// verify email exists
	// Find user by email
	var user = models.User{Email: tokenDetails.Email}
	userAndRole, err := user.FindByEmail(h.dbx)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.
				WithField("email", tokenDetails.Email).
				Error("User not found")
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_credentials",
				Message: "User not found",
			})
			return
		}
		h.logger.
			WithError(err).
			WithField("email", tokenDetails.Email).
			Error("Failed to find user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to find user",
		})
		return
	}
	user = userAndRole.User

	// 2. Update password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("user_id", user.ID).
			Error("Failed to process password")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to process password",
		})
		return
	}
	user.PasswordHash = string(hashedPassword)
	err = user.UpdatePassword(h.dbx)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("user_id", user.ID).
			Error("Failed to update password")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to update password",
		})
		return
	}
	// 3. Invalidate token
	err = h.redis.Delete(context.Background(), req.Token)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("reset_token", req.Token).
			Error("Failed to invalidate token")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to invalidate token",
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Password has been reset successfully",
	})
}

type TokenClaimsRequirement struct {
	UserID       uuid.UUID `json:"user_id"`
	SchoolID     uuid.UUID `json:"school_id"`
	RoleID       uuid.UUID `json:"role_id"`
	Email        string    `json:"email"`
	IsSuperAdmin bool      `json:"is_super_admin"`
	Exp          int64     `json:"exp"`
	Iat          int64     `json:"iat"`
}

// generateToken creates a JWT token for the user
func (h *Handler) generateToken(claims TokenClaimsRequirement) (string, error) {
	if claims.Exp == 0 {
		claims.Exp = time.Now().Add(time.Hour * time.Duration(h.config.JWTExpiryHours)).Unix()
	}
	if claims.Iat == 0 {
		claims.Iat = time.Now().Unix()
	}

	claimsJWT := jwt.MapClaims{
		"user_id":        claims.UserID.String(),
		"school_id":      claims.SchoolID.String(),
		"role_id":        claims.RoleID.String(),
		"email":          claims.Email,
		"is_super_admin": claims.IsSuperAdmin,
		"exp":            claims.Exp,
		"iat":            claims.Iat,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsJWT)
	return token.SignedString([]byte(h.config.JWTSecret))
}

type TokenClaims struct {
	TokenClaimsRequirement
	jwt.RegisteredClaims
}

// ValidateVerificationToken validates the token string and returns the email if valid.
func (h *Handler) GetTokenDetails(tokenString string) (TokenClaimsRequirement, error) {
	claims := &TokenClaims{}

	// Parse the token, automatically checking the signature and expiration time (TTL)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify that the signing method is HMAC (required by the signature validation)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key used for validation
		return []byte(h.config.JWTSecret), nil
	})

	if err != nil {
		// If the token is invalid (bad signature) OR expired (TTL check failed), an error is returned here.
		// Common errors include: "token is expired" or "signature is invalid".
		return TokenClaimsRequirement{}, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return TokenClaimsRequirement{}, fmt.Errorf("token is invalid")
	}

	// TTL check is handled automatically by jwt.ParseWithClaims()
	return TokenClaimsRequirement{
		UserID:       claims.UserID,
		SchoolID:     claims.SchoolID,
		RoleID:       claims.RoleID,
		Email:        claims.Email,
		IsSuperAdmin: claims.IsSuperAdmin,
		Exp:          claims.Exp,
		Iat:          claims.Iat,
	}, nil
}
