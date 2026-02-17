package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user (school staff) in the system
type User struct {
	BaseModel
	SchoolID     uuid.UUID `json:"school_id"`
	RoleID       uuid.UUID `json:"role_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose password hash
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Phone        string    `json:"phone"`
	IsActive     bool      `json:"is_active"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// UserClassAccess represents a user's access to a specific class
type UserClassAccess struct {
	BaseModel
	UserID  uuid.UUID `json:"user_id"`
	ClassID uuid.UUID `json:"class_id"`
}

// SetUserClassAccessRequest is the request body for setting a user's class access
type SetUserClassAccessRequest struct {
	ClassIDs []uuid.UUID `json:"class_ids" binding:"required"`
}

func (u *User) EmailExists(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", u.Email).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (u *User) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		INSERT INTO users (id, school_id, role_id, email, password_hash, first_name, last_name, phone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, u.ID, u.SchoolID, u.RoleID, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.Phone, u.IsActive, u.CreatedAt, u.UpdatedAt)
	return err
}

func (u *User) FindByEmail(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	err := dbx.QueryRowContext(ctx, `
		SELECT id, school_id, role_id, email, password_hash, first_name, last_name, phone, is_active
		FROM users WHERE email = $1
	`, u.Email).Scan(&u.ID, &u.SchoolID, &u.RoleID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Phone, &u.IsActive)
	return err
}

func (u *User) UpdatePassword(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3
	`, u.PasswordHash, time.Now(), u.ID)
	return err
}
