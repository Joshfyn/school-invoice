package models

import "github.com/google/uuid"

// Guardian represents a parent/guardian of students
type Guardian struct {
	BaseModel
	SchoolID  uuid.UUID `json:"school_id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
}

// CreateGuardianRequest is the request body for creating a guardian
type CreateGuardianRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2"`
	LastName  string `json:"last_name" binding:"required,min=2"`
	Phone     string `json:"phone" binding:"required"`
	Email     string `json:"email" binding:"omitempty,email"`
	Address   string `json:"address"`
}

// UpdateGuardianRequest is the request body for updating a guardian
type UpdateGuardianRequest struct {
	FirstName *string `json:"first_name,omitempty" binding:"omitempty,min=2"`
	LastName  *string `json:"last_name,omitempty" binding:"omitempty,min=2"`
	Phone     *string `json:"phone,omitempty"`
	Email     *string `json:"email,omitempty" binding:"omitempty,email"`
	Address   *string `json:"address,omitempty"`
}

// GuardianResponse is the response for guardian data
type GuardianResponse struct {
	ID        uuid.UUID          `json:"id"`
	SchoolID  uuid.UUID          `json:"school_id"`
	FirstName string             `json:"first_name"`
	LastName  string             `json:"last_name"`
	FullName  string             `json:"full_name"`
	Phone     string             `json:"phone"`
	Email     string             `json:"email"`
	Address   string             `json:"address"`
	Students  []StudentResponse  `json:"students,omitempty"`
}

func (g *Guardian) ToResponse() GuardianResponse {
	return GuardianResponse{
		ID:        g.ID,
		SchoolID:  g.SchoolID,
		FirstName: g.FirstName,
		LastName:  g.LastName,
		FullName:  g.FirstName + " " + g.LastName,
		Phone:     g.Phone,
		Email:     g.Email,
		Address:   g.Address,
	}
}

func (g *Guardian) FullName() string {
	return g.FirstName + " " + g.LastName
}

// Relationship represents the relationship between a guardian and a student
type Relationship string

const (
	RelationshipFather   Relationship = "father"
	RelationshipMother   Relationship = "mother"
	RelationshipGuardian Relationship = "guardian"
	RelationshipSponsor  Relationship = "sponsor"
	RelationshipOther    Relationship = "other"
)

// StudentGuardian represents the many-to-many relationship between students and guardians
type StudentGuardian struct {
	BaseModel
	StudentID              uuid.UUID    `json:"student_id"`
	GuardianID             uuid.UUID    `json:"guardian_id"`
	Relationship           Relationship `json:"relationship"`
	IsPrimary              bool         `json:"is_primary"`
	ReceivesNotifications  bool         `json:"receives_notifications"`
}

// LinkGuardianRequest is the request body for linking a guardian to a student
type LinkGuardianRequest struct {
	GuardianID            uuid.UUID    `json:"guardian_id" binding:"required"`
	Relationship          Relationship `json:"relationship" binding:"required,oneof=father mother guardian sponsor other"`
	IsPrimary             bool         `json:"is_primary"`
	ReceivesNotifications bool         `json:"receives_notifications"`
}

// CreateAndLinkGuardianRequest creates a guardian and links to a student in one request
type CreateAndLinkGuardianRequest struct {
	FirstName             string       `json:"first_name" binding:"required,min=2"`
	LastName              string       `json:"last_name" binding:"required,min=2"`
	Phone                 string       `json:"phone" binding:"required"`
	Email                 string       `json:"email" binding:"omitempty,email"`
	Address               string       `json:"address"`
	Relationship          Relationship `json:"relationship" binding:"required,oneof=father mother guardian sponsor other"`
	IsPrimary             bool         `json:"is_primary"`
	ReceivesNotifications bool         `json:"receives_notifications"`
}

// StudentGuardianResponse is the response for student-guardian relationship
type StudentGuardianResponse struct {
	ID                    uuid.UUID         `json:"id"`
	StudentID             uuid.UUID         `json:"student_id"`
	GuardianID            uuid.UUID         `json:"guardian_id"`
	Relationship          Relationship      `json:"relationship"`
	IsPrimary             bool              `json:"is_primary"`
	ReceivesNotifications bool              `json:"receives_notifications"`
	Guardian              *GuardianResponse `json:"guardian,omitempty"`
	Student               *StudentResponse  `json:"student,omitempty"`
}

func (sg *StudentGuardian) ToResponse() StudentGuardianResponse {
	return StudentGuardianResponse{
		ID:                    sg.ID,
		StudentID:             sg.StudentID,
		GuardianID:            sg.GuardianID,
		Relationship:          sg.Relationship,
		IsPrimary:             sg.IsPrimary,
		ReceivesNotifications: sg.ReceivesNotifications,
	}
}
