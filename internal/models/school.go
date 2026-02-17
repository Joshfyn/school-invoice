package models

import "github.com/google/uuid"

// School represents a school (tenant) in the system
type School struct {
	BaseModel
	Name      string  `json:"name"`
	Subdomain string  `json:"subdomain"`
	Phone     string  `json:"phone"`
	Email     string  `json:"email"`
	Address   string  `json:"address"`
	LogoURL   *string `json:"logo_url,omitempty"`
	IsActive  bool    `json:"is_active"`
}

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
	Name    *string `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	Phone   *string `json:"phone,omitempty"`
	Email   *string `json:"email,omitempty" binding:"omitempty,email"`
	Address *string `json:"address,omitempty"`
	LogoURL *string `json:"logo_url,omitempty"`
}

// SchoolResponse is the response for school data
type SchoolResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Subdomain string    `json:"subdomain"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
	LogoURL   *string   `json:"logo_url,omitempty"`
	IsActive  bool      `json:"is_active"`
}

func (s *School) ToResponse() SchoolResponse {
	return SchoolResponse{
		ID:        s.ID,
		Name:      s.Name,
		Subdomain: s.Subdomain,
		Phone:     s.Phone,
		Email:     s.Email,
		Address:   s.Address,
		LogoURL:   s.LogoURL,
		IsActive:  s.IsActive,
	}
}

func (s *School) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		INSERT INTO schools (id, name, subdomain, phone, email, address, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, s.ID, s.Name, s.Subdomain, s.Phone, s.Email, s.Address, s.IsActive, s.CreatedAt, s.UpdatedAt)
	return err
}
