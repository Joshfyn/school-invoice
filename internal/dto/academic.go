package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/models"
)

type CreateSessionRequest struct {
	Name      string    `json:"-"`
	StartDate string    `json:"start_date" binding:"required"`
	EndDate   string    `json:"end_date" binding:"required"`
	IsCurrent bool      `json:"is_current"`
	Start     time.Time `json:"-"`
	End       time.Time `json:"-"`
	SchoolID  uuid.UUID `json:"-"`
}

type UpdateSessionRequest struct {
	SessionID uuid.UUID `json:"-"`
	Name      string    `json:"-"`
	SchoolID  uuid.UUID `json:"-"`
	StartDate string    `json:"start_date,omitempty"`
	EndDate   string    `json:"end_date,omitempty"`
	Start     time.Time `json:"-"`
	End       time.Time `json:"-"`
}

type SetCurrentSessionRequest struct {
	SessionID uuid.UUID `json:"-"`
	SchoolID  uuid.UUID `json:"-"`
	IsCurrent bool      `json:"is_current"`
}

// CreateTermRequest is the request body for creating a term
type CreateTermRequest struct {
	SchoolID  uuid.UUID       `json:"-"`
	UserID    uuid.UUID       `json:"-"`
	SessionID uuid.UUID       `json:"session_id" binding:"required"`
	Name      models.TermType `json:"name" binding:"required"`
	SortOrder int             `json:"sort_order" binding:"required,min=1"`
	StartDate string          `json:"start_date" binding:"required"`
	EndDate   string          `json:"end_date" binding:"required"`
	Start     time.Time       `json:"-"`
	End       time.Time       `json:"-"`
}

// UpdateTermRequest is the request body for updating a term
type UpdateTermRequest struct {
	Name      models.TermType `json:"name,omitempty"`
	SortOrder int             `json:"sort_order,omitempty"`
	StartDate string          `json:"start_date,omitempty"`
	EndDate   string          `json:"end_date,omitempty"`
	Start     time.Time       `json:"-"`
	End       time.Time       `json:"-"`
	SchoolID  uuid.UUID       `json:"-"`
	UserID    uuid.UUID       `json:"-"`
	SessionID uuid.UUID       `json:"session_id,omitempty"`
	TermID    uuid.UUID       `json:"-"`
}
type SetCurrentTermRequest struct {
	TermID    uuid.UUID `json:"-"`
	SchoolID  uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"-"`
	IsCurrent bool      `json:"is_current" binding:"required"`
}

// CreateClassRequest is the request body for creating a class
type CreateClassRequest struct {
	Name      string `json:"name" binding:"required"`
	Section   string `json:"section" binding:"required"`
	SortOrder int    `json:"sort_order" binding:"required,min=1"`
}

// UpdateClassRequest is the request body for updating a class
type UpdateClassRequest struct {
	Name      *string `json:"name,omitempty"`
	Section   *string `json:"section,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}
