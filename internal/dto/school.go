package dto

import "github.com/google/uuid"

// CreateSchoolRequest is the request body for creating a school
type CreateSchoolRequest struct {
	Name      string `json:"name" binding:"required,min=2,max=255"`
	Subdomain string `json:"subdomain" binding:"required,min=3,max=50,alphanum"`
	Phone     string `json:"phone" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Address   string `json:"address" binding:"required"`
}

// UpdateSchoolRequest is the request body for updating a school
type UpdateSchoolRequest struct {
	Name              *string `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	Phone             *string `json:"phone,omitempty"`
	Email             *string `json:"email,omitempty" binding:"omitempty,email"`
	Address           *string `json:"address,omitempty"`
	LogoURL           *string `json:"logo_url,omitempty"`
	BankName          *string `json:"bank_name,omitempty"`
	BankAccountName   *string `json:"bank_account_name,omitempty"`
	BankAccountNumber *string `json:"bank_account_number,omitempty"`
}

// SchoolResponse is the response for school data
type SchoolResponse struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Subdomain         string    `json:"subdomain"`
	Phone             string    `json:"phone"`
	Email             string    `json:"email"`
	Address           string    `json:"address"`
	LogoURL           *string   `json:"logo_url,omitempty"`
	BankName          *string   `json:"bank_name,omitempty"`
	BankAccountName   *string   `json:"bank_account_name,omitempty"`
	BankAccountNumber *string   `json:"bank_account_number,omitempty"`
	IsActive          bool      `json:"is_active"`
}

