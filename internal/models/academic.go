package models

import (
	"time"

	"github.com/google/uuid"
)

// AcademicSession represents a school year (e.g., 2024/2025)
type AcademicSession struct {
	BaseModel
	SchoolID  uuid.UUID  `json:"school_id"`
	Name      string     `json:"name"` // e.g., "2024/2025"
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	IsCurrent *bool      `json:"is_current"`
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

func (a *AcademicSession) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		INSERT INTO academic_sessions (id, school_id, name, start_date, end_date, is_current)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, a.ID, a.SchoolID, a.Name, a.StartDate, a.EndDate, a.IsCurrent)
	return err
}

func (a *AcademicSession) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()
	// build update query
	query := "UPDATE academic_sessions SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2
	if a.Name != "" {
		query += ", name = $" + string(rune('0'+argIndex))
		args = append(args, a.Name)
		argIndex++
	}
	if a.StartDate != nil {
		query += ", start_date = $" + string(rune('0'+argIndex))
		args = append(args, a.StartDate)
		argIndex++
	}
	if a.EndDate != nil {
		query += ", end_date = $" + string(rune('0'+argIndex))
		args = append(args, a.EndDate)
		argIndex++
	}
	if a.IsCurrent != nil {
		query += ", is_current = $" + string(rune('0'+argIndex))
		args = append(args, a.IsCurrent)
		argIndex++
	}
	query += " WHERE id = $" + string(rune('0'+argIndex))
	args = append(args, a.ID)
	argIndex++
	query += " AND school_id = $" + string(rune('0'+argIndex))
	args = append(args, a.SchoolID)
	argIndex++

	_, err := dbx.ExecContext(ctx, query, args...)
	return err
}

func (a *AcademicSession) List(dbx DBTX) ([]AcademicSession, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	rows, err := dbx.QueryxContext(ctx, `
		SELECT id, school_id, name, start_date, end_date, is_current FROM academic_sessions WHERE school_id = $1
	`, a.SchoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []AcademicSession{}
	for rows.Next() {
		var session AcademicSession
		err := rows.Scan(&session.ID, &session.SchoolID, &session.Name, &session.StartDate, &session.EndDate, &session.IsCurrent)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}
