package models

import (
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
	BaseModel
	SchoolID      uuid.UUID `json:"school_id"`
	AdmissionNo   string    `json:"admission_no"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Gender        Gender    `json:"gender"`
	DateOfBirth   time.Time `json:"date_of_birth"`
	AdmissionDate time.Time `json:"admission_date"`
}

// CreateStudentRequest is the request body for creating a student
type CreateStudentRequest struct {
	AdmissionNo   string    `json:"admission_no" binding:"required"`
	FirstName     string    `json:"first_name" binding:"required,min=2"`
	LastName      string    `json:"last_name" binding:"required,min=2"`
	Gender        Gender    `json:"gender" binding:"required,oneof=male female"`
	DateOfBirth   string    `json:"date_of_birth" binding:"required"` // Format: YYYY-MM-DD
	AdmissionDate string    `json:"admission_date"`                   // Format: YYYY-MM-DD, defaults to today
	ClassID       uuid.UUID `json:"class_id" binding:"required"`      // Initial class enrollment
}

// UpdateStudentRequest is the request body for updating a student
type UpdateStudentRequest struct {
	FirstName   *string `json:"first_name,omitempty" binding:"omitempty,min=2"`
	LastName    *string `json:"last_name,omitempty" binding:"omitempty,min=2"`
	Gender      *Gender `json:"gender,omitempty" binding:"omitempty,oneof=male female"`
	DateOfBirth *string `json:"date_of_birth,omitempty"`
}

// StudentResponse is the response for student data
type StudentResponse struct {
	ID              uuid.UUID          `json:"id"`
	SchoolID        uuid.UUID          `json:"school_id"`
	AdmissionNo     string             `json:"admission_no"`
	FirstName       string             `json:"first_name"`
	LastName        string             `json:"last_name"`
	FullName        string             `json:"full_name"`
	Gender          Gender             `json:"gender"`
	DateOfBirth     string             `json:"date_of_birth"`
	AdmissionDate   string             `json:"admission_date"`
	CurrentClass    *ClassResponse     `json:"current_class,omitempty"`
	//CurrentEnrollment *EnrollmentResponse `json:"current_enrollment,omitempty"`
}

func (s *Student) ToResponse() StudentResponse {
	return StudentResponse{
		ID:            s.ID,
		SchoolID:      s.SchoolID,
		AdmissionNo:   s.AdmissionNo,
		FirstName:     s.FirstName,
		LastName:      s.LastName,
		FullName:      s.FirstName + " " + s.LastName,
		Gender:        s.Gender,
		DateOfBirth:   s.DateOfBirth.Format("2006-01-02"),
		AdmissionDate: s.AdmissionDate.Format("2006-01-02"),
	}
}

func (s *Student) FullName() string {
	return s.FirstName + " " + s.LastName
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

