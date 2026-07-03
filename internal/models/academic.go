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
	SchoolID  uuid.UUID  `json:"school_id"`
	SessionID uuid.UUID  `json:"session_id"`
	Name      TermType   `json:"name"` // e.g., FirstTerm, SecondTerm, ThirdTerm
	SortOrder int        `json:"sort_order"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	IsCurrent *bool      `json:"is_current"`
}

// Class represents a class with section (e.g., JSS1-A)
type Class struct {
	BaseModel
	SchoolID  uuid.UUID `json:"school_id" db:"school_id"`
	Name      string    `json:"name" db:"name"`
	Section   string    `json:"section" db:"section"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	IsActive  bool      `json:"is_active" db:"is_active"`
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

func (t *Term) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	// build update query
	query := "UPDATE terms SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2
	if t.Name != "" {
		query += ", name = $" + string(rune('0'+argIndex))
		args = append(args, t.Name)
		argIndex++
	}
	if t.SortOrder != 0 {
		query += ", sort_order = $" + string(rune('0'+argIndex))
		args = append(args, t.SortOrder)
		argIndex++
	}
	if t.StartDate != nil {
		query += ", start_date = $" + string(rune('0'+argIndex))
		args = append(args, t.StartDate)
		argIndex++
	}
	if t.EndDate != nil {
		query += ", end_date = $" + string(rune('0'+argIndex))
		args = append(args, t.EndDate)
		argIndex++
	}
	if t.IsCurrent != nil {
		query += ", is_current = $" + string(rune('0'+argIndex))
		args = append(args, *t.IsCurrent)
		argIndex++
	}
	query += " WHERE id = $" + string(rune('0'+argIndex))
	args = append(args, t.ID)
	argIndex++
	query += " AND school_id = $" + string(rune('0'+argIndex))
	args = append(args, t.SchoolID)
	argIndex++
	if t.SessionID != uuid.Nil {
		query += " AND session_id = $" + string(rune('0'+argIndex))
		args = append(args, t.SessionID)
		argIndex++
	}
	_, err := dbx.ExecContext(ctx, query, args...)
	return err
}

func (t *Term) List(dbx DBTX) ([]Term, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	rows, err := dbx.QueryxContext(ctx, `
		SELECT id, school_id, session_id, name, sort_order, start_date, end_date, is_current FROM terms WHERE school_id = $1
	`, t.SchoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	terms := []Term{}
	for rows.Next() {
		var term Term
		err := rows.Scan(&term.ID, &term.SchoolID, &term.SessionID, &term.Name, &term.SortOrder, &term.StartDate, &term.EndDate, &term.IsCurrent)
		if err != nil {
			return nil, err
		}
		terms = append(terms, term)
	}
	return terms, nil
}

// ClearCurrentTerm unsets is_current on all terms for a school (before marking a new current term).
func ClearCurrentTerm(dbx DBTX, schoolID uuid.UUID) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		UPDATE terms SET is_current = false, updated_at = $1
		WHERE school_id = $2 AND is_current = true
	`, time.Now().UTC(), schoolID)
	return err
}

type ClassName string

const (
	JSS1 ClassName = "JSS1"
	JSS2 ClassName = "JSS2"
	JSS3 ClassName = "JSS3"
	SS1  ClassName = "SS1"
	SS2  ClassName = "SS2"
	SS3  ClassName = "SS3"
)

// Values return a list term types
func (C *ClassName) Values() []ClassName {
	return []ClassName{
		JSS1,
		JSS2,
		JSS3,
		SS1,
		SS2,
		SS3,
	}
}

// Scan implement sql.Scanner interface
func (C *ClassName) Scan(src interface{}) error {
	var strClassName string
	switch v := src.(type) {
	case string:
		strClassName = v
	case []uint8:
		strClassName = string(v)
	case TermType:
		strClassName = string(v)
	default:
		return fmt.Errorf("incompatible type %T for class name %v", src, src)
	}

	switch strClassName {
	case string(FirstTerm):
		*C = JSS1
	case string(JSS2):
		*C = JSS2
	case string(JSS3):
		*C = JSS3
	case string(SS1):
		*C = SS1
	case string(SS2):
		*C = SS2
	case string(SS3):
		*C = SS3
	default:
		return fmt.Errorf("class name %s not supported", strClassName)
	}

	return nil
}

// IsValid validate whether ConversionType is correct
func (C ClassName) IsValid() bool {
	switch C {
	case JSS1,
		JSS2,
		JSS3,
		SS1,
		SS2,
		SS3:
		return true
	default:
		return false
	}
}

// String return a string value of a conversion type
func (C ClassName) String() string {
	return string(C)
}

func (c *Class) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	_, err := dbx.ExecContext(ctx, `
		INSERT INTO classes (id, school_id, name, section, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, c.ID, c.SchoolID, c.Name, c.Section, c.SortOrder, c.IsActive)
	return err
}

func (c *Class) Get(dbx DBTX, schoolID uuid.UUID) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	return dbx.GetContext(ctx, c, `
		SELECT id, school_id, name, section, sort_order, is_active, created_at, updated_at
		FROM classes
		WHERE id = $1 AND school_id = $2
	`, c.ID, schoolID)
}

func (c *Class) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	query := "UPDATE classes SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2

	if c.Name != "" {
		query += fmt.Sprintf(", name = $%d", argIndex)
		args = append(args, c.Name)
		argIndex++
	}
	if c.Section != "" {
		query += fmt.Sprintf(", section = $%d", argIndex)
		args = append(args, c.Section)
		argIndex++
	}
	if c.SortOrder > 0 {
		query += fmt.Sprintf(", sort_order = $%d", argIndex)
		args = append(args, c.SortOrder)
		argIndex++
	}
	query += fmt.Sprintf(", is_active = $%d", argIndex)
	args = append(args, c.IsActive)
	argIndex++

	query += fmt.Sprintf(" WHERE id = $%d AND school_id = $%d", argIndex, argIndex+1)
	args = append(args, c.ID, c.SchoolID)

	_, err := dbx.ExecContext(ctx, query, args...)
	return err
}

func ListClasses(dbx DBTX, schoolID uuid.UUID, level string) ([]Class, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	query := `
		SELECT id, school_id, name, section, sort_order, is_active, created_at, updated_at
		FROM classes
		WHERE school_id = $1
	`
	args := []interface{}{schoolID}
	if level != "" {
		query += " AND name = $2"
		args = append(args, level)
	}
	query += " ORDER BY sort_order, name, section"

	classes := []Class{}
	if err := dbx.SelectContext(ctx, &classes, query, args...); err != nil {
		return nil, err
	}
	return classes, nil
}

func (c *Class) ToResponse() ClassResponse {
	return ClassResponse{
		ID:          c.ID,
		SchoolID:    c.SchoolID,
		Name:        c.Name,
		Section:     c.Section,
		DisplayName: c.DisplayName(),
		SortOrder:   c.SortOrder,
		IsActive:    c.IsActive,
	}
}
