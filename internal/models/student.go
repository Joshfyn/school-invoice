package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Gender represents the gender of a student
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

// Student represents a student in a school
type Student struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	FirstName   string     `json:"first_name" db:"first_name"`
	MiddleName  string     `json:"middle_name" db:"middle_name"`
	LastName    string     `json:"last_name" db:"last_name"`
	Gender      Gender     `json:"gender" db:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth" db:"date_of_birth"`
	NIN         string     `json:"nin" db:"nin"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// EnrollmentStatus represents the status of a student enrollment
type EnrollmentStatus string

const (
	EnrollmentActive    EnrollmentStatus = "active"
	EnrollmentPromoted  EnrollmentStatus = "promoted"
	EnrollmentWithdrawn EnrollmentStatus = "withdrawn"
)

// StudentEnrollment represents a student's enrollment in a class for a term
type StudentEnrollment struct {
	BaseModel
	SchoolID   uuid.UUID        `json:"school_id"`
	StudentID  uuid.UUID        `json:"student_id"`
	ClassID    uuid.UUID        `json:"class_id"`
	TermID     uuid.UUID        `json:"term_id"`
	Status     EnrollmentStatus `json:"status"`
	EnrolledAt time.Time        `json:"enrolled_at"`
}

// CreateEnrollmentRequest is the request body for creating an enrollment
type CreateEnrollmentRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	ClassID   uuid.UUID `json:"class_id" binding:"required"`
	TermID    uuid.UUID `json:"term_id" binding:"required"`
}

// BulkEnrollmentRequest is the request body for bulk enrollment (e.g., promoting a class)
type BulkEnrollmentRequest struct {
	FromClassID uuid.UUID   `json:"from_class_id" binding:"required"`
	ToClassID   uuid.UUID   `json:"to_class_id" binding:"required"`
	TermID      uuid.UUID   `json:"term_id" binding:"required"`
	StudentIDs  []uuid.UUID `json:"student_ids,omitempty"` // If empty, promote all students in from_class
}

// UpdateEnrollmentStatusRequest is the request body for updating enrollment status
type UpdateEnrollmentStatusRequest struct {
	Status EnrollmentStatus `json:"status" binding:"required,oneof=active promoted withdrawn"`
}

func (s *Student) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	rows, err := NamedQueryContext(dbx, ctx, `
		INSERT INTO students (first_name, middle_name, last_name, gender, date_of_birth, nin)
		VALUES (:first_name, :middle_name, :last_name, :gender, :date_of_birth, :nin) 
		RETURNING id, first_name, middle_name, last_name, gender, date_of_birth, nin, created_at, updated_at
	`, s)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return errors.New("student not created")
	}
	if err := rows.StructScan(s); err != nil {
		return err
	}
	for rows.Next() {
		if err := rows.StructScan(s); err != nil {
			return err
		}
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	return nil
}

func (s *Student) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	query := "UPDATE students SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2
	if s.FirstName != "" {
		query += ", first_name = $" + string(rune('0'+argIndex))
		args = append(args, s.FirstName)
		argIndex++
	}
	if s.MiddleName != "" {
		query += ", middle_name = $" + string(rune('0'+argIndex))
		args = append(args, s.MiddleName)
		argIndex++
	}
	if s.LastName != "" {
		query += ", last_name = $" + string(rune('0'+argIndex))
		args = append(args, s.LastName)
		argIndex++
	}
	if s.Gender != "" {
		query += ", gender = $" + string(rune('0'+argIndex))
		args = append(args, s.Gender)
		argIndex++
	}
	if s.DateOfBirth != nil {
		query += ", date_of_birth = $" + string(rune('0'+argIndex))
		args = append(args, s.DateOfBirth)
		argIndex++
	}
	if s.NIN != "" {
		query += ", nin = $" + string(rune('0'+argIndex))
		args = append(args, s.NIN)
		argIndex++
	}
	query += " WHERE id = $" + string(rune('0'+argIndex))
	args = append(args, s.ID)
	argIndex++
	query += " AND deleted_at IS NULL"
	student, err := dbx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	studentRows, err := student.RowsAffected()
	if err != nil {
		return err
	}
	if studentRows == 0 {
		return errors.New("student not found")
	}
	return nil
}

func (s *Student) Get(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	err := dbx.QueryRowContext(ctx, "SELECT id, nin, first_name, middle_name, last_name, gender, date_of_birth FROM students WHERE id = $1", s.ID).
		Scan(&s.ID, &s.NIN, &s.FirstName, &s.MiddleName, &s.LastName, &s.Gender, &s.DateOfBirth)
	if err != nil {
		return err
	}
	return nil
}

func (s *Student) NINExists(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM students WHERE nin = $1)", s.NIN).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
