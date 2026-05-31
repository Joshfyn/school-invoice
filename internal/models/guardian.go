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
	StudentID             uuid.UUID    `json:"student_id" db:"student_id"`
	GuardianID            uuid.UUID    `json:"guardian_id" db:"guardian_id"`
	Relationship          Relationship `json:"relationship" db:"relationship"`
	IsPrimary             bool         `json:"is_primary" db:"is_primary"`
	ReceivesNotifications bool         `json:"receives_notifications" db:"receives_notifications"`
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

	err := dbx.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, phone, email, address, created_at, updated_at
		FROM guardians WHERE id = $1 AND deleted_at IS NULL
	`, g.ID).
		Scan(&g.ID, &g.FirstName, &g.LastName, &g.Phone, &g.Email, &g.Address, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (g *Guardian) List(dbx DBTX, schoolID uuid.UUID) ([]Guardian, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	guardians := []Guardian{}
	err := dbx.SelectContext(ctx, &guardians, `
		SELECT DISTINCT g.id, g.first_name, g.last_name, g.phone, g.email, g.address, g.created_at, g.updated_at
		FROM guardians g
		INNER JOIN student_guardians sg ON sg.guardian_id = g.id
		INNER JOIN student_admission sa ON sa.student_id = sg.student_id AND sa.deleted_at IS NULL
		WHERE sa.school_id = $1 AND g.deleted_at IS NULL
		ORDER BY g.last_name, g.first_name
	`, schoolID)
	return guardians, err
}

func ListAllGuardians(dbx DBTX) ([]Guardian, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	guardians := []Guardian{}
	err := dbx.SelectContext(ctx, &guardians, `
		SELECT id, first_name, last_name, phone, email, address, created_at, updated_at
		FROM guardians
		WHERE deleted_at IS NULL
		ORDER BY last_name, first_name
	`)
	return guardians, err
}

func (g *Guardian) PhoneExists(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM guardians WHERE phone = $1 AND deleted_at IS NULL)", g.Phone).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (g *Guardian) PhoneExistsForOther(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM guardians WHERE phone = $1 AND id != $2 AND deleted_at IS NULL)",
		g.Phone, g.ID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (g *Guardian) EmailExists(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM guardians WHERE email = $1 AND deleted_at IS NULL)", g.Email).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (g *Guardian) EmailExistsForOther(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM guardians WHERE email = $1 AND id != $2 AND email != '' AND deleted_at IS NULL)",
		g.Email, g.ID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

type GuardianStudentSummary struct {
	ID           uuid.UUID    `json:"id"`
	FirstName    string       `json:"first_name"`
	LastName     string       `json:"last_name"`
	Relationship Relationship `json:"relationship"`
}

type GuardianListItem struct {
	Guardian
	Students []GuardianStudentSummary `json:"students"`
}

func ListGuardiansForSchool(dbx DBTX, schoolID uuid.UUID) ([]GuardianListItem, error) {
	guardians, err := (&Guardian{}).List(dbx, schoolID)
	if err != nil {
		return nil, err
	}
	if len(guardians) == 0 {
		return []GuardianListItem{}, nil
	}

	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	rows, err := dbx.QueryxContext(ctx, `
		SELECT sg.guardian_id, sg.relationship, s.id, s.first_name, s.last_name
		FROM student_guardians sg
		INNER JOIN students s ON s.id = sg.student_id
		INNER JOIN student_admission sa ON sa.student_id = s.id AND sa.deleted_at IS NULL
		WHERE sa.school_id = $1 AND s.deleted_at IS NULL
		ORDER BY s.last_name, s.first_name
	`, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	studentsByGuardian := map[uuid.UUID][]GuardianStudentSummary{}
	for rows.Next() {
		var guardianID, studentID uuid.UUID
		var relationship Relationship
		var firstName, lastName string
		if err := rows.Scan(&guardianID, &relationship, &studentID, &firstName, &lastName); err != nil {
			return nil, err
		}
		studentsByGuardian[guardianID] = append(studentsByGuardian[guardianID], GuardianStudentSummary{
			ID:           studentID,
			FirstName:    firstName,
			LastName:     lastName,
			Relationship: relationship,
		})
	}

	items := make([]GuardianListItem, 0, len(guardians))
	for _, g := range guardians {
		students := studentsByGuardian[g.ID]
		if students == nil {
			students = []GuardianStudentSummary{}
		}
		items = append(items, GuardianListItem{Guardian: g, Students: students})
	}
	return items, nil
}

func StudentAdmittedToSchool(dbx DBTX, schoolID, studentID uuid.UUID) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM student_admission
			WHERE student_id = $1 AND school_id = $2 AND deleted_at IS NULL
		)
	`, studentID, schoolID).Scan(&exists)
	return exists, err
}

type GuardianDetailResponse struct {
	Guardian
	Students []StudentGuardianResponse `json:"students"`
}

type StudentGuardianResponse struct {
	StudentGuardian
	Guardian *Guardian `json:"guardian,omitempty"`
	Student  *Student  `json:"student,omitempty"`
}

func (sg *StudentGuardian) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if sg.ID == uuid.Nil {
		sg.ID = uuid.New()
	}

	rows, err := NamedQueryContext(dbx, ctx, `
		INSERT INTO student_guardians (id, student_id, guardian_id, relationship, is_primary, receives_notifications)
		VALUES (:id, :student_id, :guardian_id, :relationship, :is_primary, :receives_notifications)
		RETURNING id, student_id, guardian_id, relationship, is_primary, receives_notifications, created_at, updated_at
	`, sg)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("student guardian link not created")
	}
	return rows.StructScan(sg)
}

func ListStudentGuardians(dbx DBTX, studentID uuid.UUID) ([]StudentGuardianResponse, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	rows, err := dbx.QueryxContext(ctx, `
		SELECT sg.id, sg.student_id, sg.guardian_id, sg.relationship, sg.is_primary, sg.receives_notifications,
		       sg.created_at, sg.updated_at,
		       g.id, g.first_name, g.last_name, g.phone, g.email, g.address
		FROM student_guardians sg
		INNER JOIN guardians g ON g.id = sg.guardian_id
		WHERE sg.student_id = $1 AND g.deleted_at IS NULL
		ORDER BY sg.is_primary DESC, g.last_name
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []StudentGuardianResponse{}
	for rows.Next() {
		var resp StudentGuardianResponse
		var guardian Guardian
		if err := rows.Scan(
			&resp.ID, &resp.StudentID, &resp.GuardianID, &resp.Relationship, &resp.IsPrimary, &resp.ReceivesNotifications,
			&resp.CreatedAt, &resp.UpdatedAt,
			&guardian.ID, &guardian.FirstName, &guardian.LastName, &guardian.Phone, &guardian.Email, &guardian.Address,
		); err != nil {
			return nil, err
		}
		resp.Guardian = &guardian
		results = append(results, resp)
	}
	return results, nil
}

func GetGuardianWithStudents(dbx DBTX, schoolID, guardianID uuid.UUID) (*GuardianDetailResponse, error) {
	guardian := Guardian{ID: guardianID}
	if err := guardian.Get(dbx); err != nil {
		return nil, err
	}

	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	rows, err := dbx.QueryxContext(ctx, `
		SELECT sg.id, sg.student_id, sg.guardian_id, sg.relationship, sg.is_primary, sg.receives_notifications,
		       sg.created_at, sg.updated_at,
		       s.id, s.first_name, s.middle_name, s.last_name, s.gender, s.date_of_birth, s.nin
		FROM student_guardians sg
		INNER JOIN students s ON s.id = sg.student_id
		INNER JOIN student_admission sa ON sa.student_id = s.id AND sa.deleted_at IS NULL
		WHERE sg.guardian_id = $1 AND sa.school_id = $2 AND s.deleted_at IS NULL
	`, guardianID, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	detail := &GuardianDetailResponse{Guardian: guardian}
	for rows.Next() {
		var link StudentGuardianResponse
		var student Student
		if err := rows.Scan(
			&link.ID, &link.StudentID, &link.GuardianID, &link.Relationship, &link.IsPrimary, &link.ReceivesNotifications,
			&link.CreatedAt, &link.UpdatedAt,
			&student.ID, &student.FirstName, &student.MiddleName, &student.LastName, &student.Gender, &student.DateOfBirth, &student.NIN,
		); err != nil {
			return nil, err
		}
		link.Student = &student
		detail.Students = append(detail.Students, link)
	}
	return detail, nil
}
