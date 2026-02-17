package models

import (
	"time"

	"github.com/google/uuid"
)

// AcademicSession represents a school year (e.g., 2024/2025)
type AcademicSession struct {
	BaseModel
	SchoolID  uuid.UUID `json:"school_id"`
	Name      string    `json:"name"` // e.g., "2024/2025"
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsCurrent bool      `json:"is_current"`
}

// CreateSessionRequest is the request body for creating an academic session
type CreateSessionRequest struct {
	Name      string `json:"name" binding:"required"`
	StartDate string `json:"start_date" binding:"required"` // Format: YYYY-MM-DD
	EndDate   string `json:"end_date" binding:"required"`   // Format: YYYY-MM-DD
}

// UpdateSessionRequest is the request body for updating an academic session
type UpdateSessionRequest struct {
	Name      *string `json:"name,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
}

// SessionResponse is the response for session data
type SessionResponse struct {
	ID        uuid.UUID `json:"id"`
	SchoolID  uuid.UUID `json:"school_id"`
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	IsCurrent bool      `json:"is_current"`
}

func (s *AcademicSession) ToResponse() SessionResponse {
	return SessionResponse{
		ID:        s.ID,
		SchoolID:  s.SchoolID,
		Name:      s.Name,
		StartDate: s.StartDate.Format("2006-01-02"),
		EndDate:   s.EndDate.Format("2006-01-02"),
		IsCurrent: s.IsCurrent,
	}
}

// Term represents a term within an academic session
type Term struct {
	BaseModel
	SchoolID  uuid.UUID `json:"school_id"`
	SessionID uuid.UUID `json:"session_id"`
	Name      string    `json:"name"` // e.g., "First Term", "Second Term", "Third Term"
	SortOrder int       `json:"sort_order"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsCurrent bool      `json:"is_current"`
}

// CreateTermRequest is the request body for creating a term
type CreateTermRequest struct {
	SessionID uuid.UUID `json:"session_id" binding:"required"`
	Name      string    `json:"name" binding:"required"`
	SortOrder int       `json:"sort_order" binding:"required,min=1"`
	StartDate string    `json:"start_date" binding:"required"`
	EndDate   string    `json:"end_date" binding:"required"`
}

// UpdateTermRequest is the request body for updating a term
type UpdateTermRequest struct {
	Name      *string `json:"name,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
}

// TermResponse is the response for term data
type TermResponse struct {
	ID        uuid.UUID        `json:"id"`
	SchoolID  uuid.UUID        `json:"school_id"`
	SessionID uuid.UUID        `json:"session_id"`
	Name      string           `json:"name"`
	SortOrder int              `json:"sort_order"`
	StartDate string           `json:"start_date"`
	EndDate   string           `json:"end_date"`
	IsCurrent bool             `json:"is_current"`
	Session   *SessionResponse `json:"session,omitempty"`
}

func (t *Term) ToResponse() TermResponse {
	return TermResponse{
		ID:        t.ID,
		SchoolID:  t.SchoolID,
		SessionID: t.SessionID,
		Name:      t.Name,
		SortOrder: t.SortOrder,
		StartDate: t.StartDate.Format("2006-01-02"),
		EndDate:   t.EndDate.Format("2006-01-02"),
		IsCurrent: t.IsCurrent,
	}
}

// Class represents a class with section (e.g., JSS1-A)
type Class struct {
	BaseModel
	SchoolID  uuid.UUID `json:"school_id"`
	Name      string    `json:"name"`    // e.g., "JSS1", "SS2"
	Section   string    `json:"section"` // e.g., "A", "B", "C"
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
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

// ClassResponse is the response for class data
type ClassResponse struct {
	ID          uuid.UUID `json:"id"`
	SchoolID    uuid.UUID `json:"school_id"`
	Name        string    `json:"name"`
	Section     string    `json:"section"`
	DisplayName string    `json:"display_name"` // e.g., "JSS1-A"
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
}

func (c *Class) ToResponse() ClassResponse {
	return ClassResponse{
		ID:          c.ID,
		SchoolID:    c.SchoolID,
		Name:        c.Name,
		Section:     c.Section,
		DisplayName: c.Name + "-" + c.Section,
		SortOrder:   c.SortOrder,
		IsActive:    c.IsActive,
	}
}

func (c *Class) DisplayName() string {
	return c.Name + "-" + c.Section
}
