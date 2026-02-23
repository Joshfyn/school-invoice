package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TermType string

const (
	FirstTerm  TermType = "first"
	SecondTerm TermType = "second"
	ThirdTerm  TermType = "third"
)

// Values return a list term types
func (TT *TermType) Values() []TermType {
	return []TermType{
		FirstTerm,
		SecondTerm,
		ThirdTerm,
	}
}

// Scan implement sql.Scanner interface
func (TT *TermType) Scan(src interface{}) error {
	var strTermType string
	switch v := src.(type) {
	case string:
		strTermType = v
	case []uint8:
		strTermType = string(v)
	case TermType:
		strTermType = string(v)
	default:
		return fmt.Errorf("incompatible type %T for term type %v", src, src)
	}

	switch strTermType {
	case string(FirstTerm):
		*TT = FirstTerm
	case string(SecondTerm):
		*TT = SecondTerm
	case string(ThirdTerm):
		*TT = ThirdTerm
	default:
		return fmt.Errorf("term type %s not supported", strTermType)
	}

	return nil
}

// IsValid validate whether ConversionType is correct
func (TT TermType) IsValid() bool {
	switch TT {
	case FirstTerm,
		SecondTerm,
		ThirdTerm:
		return true
	default:
		return false
	}
}

// String return a string value of a conversion type
func (TT TermType) String() string {
	return string(TT)
}

// AcademicSession represents a school year (e.g., 2024/2025)
type AcademicSession struct {
	BaseModel
	SchoolID  uuid.UUID  `json:"school_id"`
	Name      string     `json:"name"` // e.g., "2024/2025"
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	IsCurrent *bool      `json:"is_current"`
}

// Term represents a term within an academic session
type Term struct {
	BaseModel
	SchoolID  uuid.UUID `json:"school_id"`
	SessionID uuid.UUID `json:"session_id"`
	Name      TermType  `json:"name"` // e.g., FirstTerm, SecondTerm, ThirdTerm
	SortOrder int       `json:"sort_order"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsCurrent bool      `json:"is_current"`
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

func (t *Term) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		INSERT INTO terms (id, school_id, session_id, name, sort_order, start_date, end_date, is_current)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, t.ID, t.SchoolID, t.SessionID, t.Name, t.SortOrder, t.StartDate, t.EndDate, t.IsCurrent)
	return err
}
