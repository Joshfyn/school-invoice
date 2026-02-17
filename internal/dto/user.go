package dto

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Token       string `json:"-"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
