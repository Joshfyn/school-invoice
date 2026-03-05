package dto

import "github.com/google/uuid"

type ListGuardiansRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Search   string `json:"search"`
	Sort     string `json:"sort"`
	SortDir  string `json:"sort_dir"`
}

// CreateGuardianRequest is the request body for creating a guardian
type CreateGuardianRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2"`
	LastName  string `json:"last_name" binding:"required,min=2"`
	Phone     string `json:"phone" binding:"required,phone"`
	Email     string `json:"email" binding:"omitempty,email"`
	Address   string `json:"address,omitempty"`
	Nin       string `json:"nin,omitempty"`
}

type UpdateGuardianRequest struct {
	GuardianID uuid.UUID `json:"-"`
	FirstName  string    `json:"first_name,omitempty" binding:"omitempty,min=2"`
	LastName   string    `json:"last_name,omitempty" binding:"omitempty,min=2"`
	Phone      string    `json:"phone,omitempty"`
	Email      string    `json:"email,omitempty" binding:"omitempty,email"`
	Address    string    `json:"address,omitempty"`
}
