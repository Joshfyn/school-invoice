package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Guardian represents a parent/guardian of students
type Guardian struct {
	ID        uuid.UUID `json:"id" db:"id"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	Phone     string    `json:"phone" db:"phone"`
	Email     string    `json:"email" db:"email"`
	Address   string    `json:"address" db:"address"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
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
	StudentID             uuid.UUID    `json:"student_id"`
	GuardianID            uuid.UUID    `json:"guardian_id"`
	Relationship          Relationship `json:"relationship"`
	IsPrimary             bool         `json:"is_primary"`
	ReceivesNotifications bool         `json:"receives_notifications"`
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

func (G *Guardian) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	rows, err := NamedQueryContext(dbx, ctx, `INSERT INTO guardians (first_name, last_name, phone, email, address) 
										VALUES
										 (:first_name, :last_name, 
										 :phone, :email, :address) RETURNING id, first_name,
										  last_name, phone, email, address, created_at, updated_at`, G)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return errors.New("guardian not created")
	}
	if err := rows.StructScan(&G); err != nil {
		return err
	}
	for rows.Next() {
		if err := rows.StructScan(&G); err != nil {
			return err
		}
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	return nil
}

func (g *Guardian) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()
	query := "UPDATE guardians SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2
	if g.FirstName != "" {
		query += ", first_name = $" + string(rune('0'+argIndex))
		args = append(args, g.FirstName)
		argIndex++
	}
	if g.LastName != "" {
		query += ", last_name = $" + string(rune('0'+argIndex))
		args = append(args, g.LastName)
		argIndex++
	}
	if g.Phone != "" {
		query += ", phone = $" + string(rune('0'+argIndex))
		args = append(args, g.Phone)
		argIndex++
	}
	if g.Email != "" {
		query += ", email = $" + string(rune('0'+argIndex))
		args = append(args, g.Email)
		argIndex++
	}
	if g.Address != "" {
		query += ", address = $" + string(rune('0'+argIndex))
		args = append(args, g.Address)
		argIndex++
	}
	query += " WHERE id = $" + string(rune('0'+argIndex))
	args = append(args, g.ID)
	argIndex++
	query += " AND deleted_at IS NULL"
	guardian, err := dbx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	guardianRows, err := guardian.RowsAffected()
	if err != nil {
		return err
	}
	if guardianRows == 0 {
		return errors.New("guardian not found")
	}
	return err
}

func (g *Guardian) Get(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	err := dbx.QueryRowContext(ctx, "SELECT id, first_name, last_name, phone, email, address FROM guardians WHERE id = $1", g.ID).
		Scan(&g.ID, &g.FirstName, &g.LastName, &g.Phone, &g.Email, &g.Address)
	if err != nil {
		return err
	}

	return nil
}

func (g *Guardian) List(dbx DBTX) ([]Guardian, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	rows, err := dbx.QueryxContext(ctx, "SELECT id, school_id, first_name, last_name, phone, email, address FROM guardians")
	if err != nil {
		return []Guardian{}, err
	}
	defer rows.Close()

	guardians := []Guardian{}
	for rows.Next() {
		var guardian Guardian
		err := rows.Scan(&guardian.ID, &guardian.FirstName, &guardian.LastName, &guardian.Phone, &guardian.Email, &guardian.Address)
		if err != nil {
			return []Guardian{}, err
		}
		guardians = append(guardians, guardian)
	}
	return guardians, nil
}

func (g *Guardian) PhoneExists(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM guardians WHERE phone = $1)", g.Phone).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (g *Guardian) EmailExists(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM guardians WHERE email = $1)", g.Email).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
