package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user (school staff) in the system
type User struct {
	BaseModel
	SchoolID     uuid.UUID `json:"school_id" db:"school_id"`
	RoleID       uuid.UUID `json:"role_id" db:"role_id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose password hash
	FirstName    string    `json:"first_name" db:"first_name"`
	LastName     string    `json:"last_name" db:"last_name"`
	Phone        string    `json:"phone" db:"phone"`
	IsActive     bool      `json:"is_active" db:"is_active"`
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

// GetUsers returns a list of users, along side the school information and role details
func GetUsers(dbx DBTX, schoolID uuid.UUID) ([]struct {
	User   `db:"user"`
	School School `db:"school"`
	Role   Role   `db:"role"`
}, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	users := []struct {
		User   `db:"user"`
		School School `db:"school"`
		Role   Role   `db:"role"`
	}{}
	err := dbx.SelectContext(ctx, &users, `
		SELECT
			u.id AS "user.id",
			u.school_id AS "user.school_id",
			u.role_id AS "user.role_id",
			u.email AS "user.email",
			u.password_hash AS "user.password_hash",
			u.first_name AS "user.first_name",
			u.last_name AS "user.last_name",
			u.phone AS "user.phone",
			u.is_active AS "user.is_active",
			u.created_at AS "user.created_at",
			u.updated_at AS "user.updated_at",

			s.id AS "school.id",
			s.name AS "school.name",
			s.subdomain AS "school.subdomain",
			s.phone AS "school.phone",
			s.email AS "school.email",
			s.address AS "school.address",
			s.logo_url AS "school.logo_url",
			s.is_active AS "school.is_active",
			s.created_at AS "school.created_at",
			s.updated_at AS "school.updated_at",

			r.id AS "role.id",
			r.school_id AS "role.school_id",
			r.name AS "role.name",
			r.description AS "role.description",
			r.permissions AS "role.permissions",
			r.is_super_admin AS "role.is_super_admin",
			r.created_at AS "role.created_at",
			r.updated_at AS "role.updated_at"
		FROM users u
		JOIN schools s ON u.school_id = s.id
		JOIN roles r ON u.role_id = r.id
		WHERE u.school_id = $1
	`, schoolID)

	return users, err
}
