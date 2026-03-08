package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/models"
)

// CreateStudentRequest is the request body for creating a student
type CreateStudentRequest struct {
	FirstName   string        `json:"first_name" binding:"required,min=2"`
	MiddleName  string        `json:"middle_name,omitempty"`
	LastName    string        `json:"last_name" binding:"required,min=2"`
	Gender      models.Gender `json:"gender" binding:"required,oneof=male female"`
	DateOfBirth *time.Time    `json:"date_of_birth" binding:"required"` // Format: YYYY-MM-DD
	NIN         string        `json:"nin" binding:"required"`
}

// UpdateStudentRequest is the request body for updating a student
type UpdateStudentRequest struct {
	StudentID   uuid.UUID     `json:"-"`
	FirstName   string        `json:"first_name,omitempty" binding:"omitempty,min=2"`
	MiddleName  string        `json:"middle_name,omitempty"`
	LastName    string        `json:"last_name,omitempty" binding:"omitempty,min=2"`
	Gender      models.Gender `json:"gender,omitempty" binding:"omitempty,oneof=male female"`
	DateOfBirth time.Time     `json:"date_of_birth,omitempty"`
	NIN         string        `json:"nin,omitempty"`
}

// GetSingleStudentRequest is the request body for getting a single student
type GetSingleStudentRequest struct {
	StudentID uuid.UUID `json:"-"`
}
