package dto

import (
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/models"
)

type CreateRoleRequest struct {
	Name        string             `json:"name" binding:"required,min=2,max=100"`
	Description string             `json:"description" binding:"max=500"`
	Permissions models.Permissions `json:"permissions" binding:"required"`
}

// UpdateRoleRequest is the request body for updating a role
type UpdateRoleRequest struct {
	RoleID      uuid.UUID           `json:"-"`
	Name        *string             `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Description *string             `json:"description,omitempty" binding:"omitempty,max=500"`
	Permissions *models.Permissions `json:"permissions,omitempty"`
}

// RoleResponse is the response for role data
type RoleResponse struct {
	ID           uuid.UUID          `json:"id"`
	SchoolID     uuid.UUID          `json:"school_id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Permissions  models.Permissions `json:"permissions"`
	IsSuperAdmin bool               `json:"is_super_admin"`
}
