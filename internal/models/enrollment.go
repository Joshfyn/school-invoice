package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type EnrollmentResponse struct {
	ID         uuid.UUID        `json:"id"`
	SchoolID   uuid.UUID        `json:"school_id"`
	StudentID  uuid.UUID        `json:"student_id"`
	ClassID    uuid.UUID        `json:"class_id"`
	TermID     uuid.UUID        `json:"term_id"`
	Status     EnrollmentStatus `json:"status"`
	EnrolledAt time.Time        `json:"enrolled_at"`
	Class      *ClassResponse   `json:"class,omitempty"`
	Term       *Term            `json:"term,omitempty"`
}

type EnrollmentFilters struct {
	TermID  *uuid.UUID
	ClassID *uuid.UUID
	Status  *EnrollmentStatus
}

func (e *StudentEnrollment) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Status == "" {
		e.Status = EnrollmentActive
	}
	if e.EnrolledAt.IsZero() {
		e.EnrolledAt = time.Now().UTC()
	}

	rows, err := NamedQueryContext(dbx, ctx, `
		INSERT INTO student_enrollments (id, school_id, student_id, class_id, term_id, status, enrolled_at)
		VALUES (:id, :school_id, :student_id, :class_id, :term_id, :status, :enrolled_at)
		RETURNING id, school_id, student_id, class_id, term_id, status, enrolled_at, created_at, updated_at
	`, e)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return errors.New("enrollment not created")
	}
	return rows.StructScan(e)
}

func (e *StudentEnrollment) Get(dbx DBTX, schoolID uuid.UUID) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	return dbx.GetContext(ctx, e, `
		SELECT id, school_id, student_id, class_id, term_id, status, enrolled_at, created_at, updated_at
		FROM student_enrollments
		WHERE id = $1 AND school_id = $2
	`, e.ID, schoolID)
}

func (e *StudentEnrollment) UpdateStatus(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	result, err := dbx.ExecContext(ctx, `
		UPDATE student_enrollments
		SET status = $1, updated_at = $2
		WHERE id = $3 AND school_id = $4
	`, e.Status, time.Now().UTC(), e.ID, e.SchoolID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func ListEnrollments(dbx DBTX, schoolID uuid.UUID, filters EnrollmentFilters) ([]StudentEnrollment, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	query := `
		SELECT id, school_id, student_id, class_id, term_id, status, enrolled_at, created_at, updated_at
		FROM student_enrollments
		WHERE school_id = $1
	`
	args := []interface{}{schoolID}
	argIndex := 2

	if filters.TermID != nil {
		query += fmt.Sprintf(" AND term_id = $%d", argIndex)
		args = append(args, *filters.TermID)
		argIndex++
	}
	if filters.ClassID != nil {
		query += fmt.Sprintf(" AND class_id = $%d", argIndex)
		args = append(args, *filters.ClassID)
		argIndex++
	}
	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filters.Status)
	}
	query += " ORDER BY enrolled_at DESC"

	enrollments := []StudentEnrollment{}
	if err := dbx.SelectContext(ctx, &enrollments, query, args...); err != nil {
		return nil, err
	}
	return enrollments, nil
}

func ListStudentEnrollments(dbx DBTX, schoolID, studentID uuid.UUID) ([]StudentEnrollment, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	enrollments := []StudentEnrollment{}
	err := dbx.SelectContext(ctx, &enrollments, `
		SELECT id, school_id, student_id, class_id, term_id, status, enrolled_at, created_at, updated_at
		FROM student_enrollments
		WHERE school_id = $1 AND student_id = $2
		ORDER BY enrolled_at DESC
	`, schoolID, studentID)
	return enrollments, err
}

func ListClassStudents(dbx DBTX, schoolID, classID uuid.UUID) ([]Student, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	students := []Student{}
	err := dbx.SelectContext(ctx, &students, `
		SELECT DISTINCT s.id, s.first_name, s.middle_name, s.last_name, s.gender, s.date_of_birth, s.nin, s.created_at, s.updated_at
		FROM students s
		INNER JOIN student_enrollments e ON e.student_id = s.id
		INNER JOIN terms t ON t.id = e.term_id
		WHERE e.school_id = $1
		  AND e.class_id = $2
		  AND e.status = 'active'
		  AND t.is_current = true
		  AND s.deleted_at IS NULL
		ORDER BY s.last_name, s.first_name
	`, schoolID, classID)
	return students, err
}

func BulkEnrollStudents(dbx DBTX, schoolID uuid.UUID, req BulkEnrollmentRequest) (int, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	studentIDs := req.StudentIDs
	if len(studentIDs) == 0 {
		err := dbx.SelectContext(ctx, &studentIDs, `
			SELECT DISTINCT e.student_id
			FROM student_enrollments e
			INNER JOIN terms t ON t.id = e.term_id
			WHERE e.school_id = $1
			  AND e.class_id = $2
			  AND e.status = 'active'
			  AND t.is_current = true
		`, schoolID, req.FromClassID)
		if err != nil {
			return 0, err
		}
	}

	count := 0
	for _, studentID := range studentIDs {
		var existingID uuid.UUID
		err := dbx.GetContext(ctx, &existingID, `
			SELECT id FROM student_enrollments
			WHERE student_id = $1 AND term_id = $2
		`, studentID, req.TermID)

		if err == nil {
			_, err = dbx.ExecContext(ctx, `
				UPDATE student_enrollments
				SET class_id = $1, status = 'active', updated_at = $2
				WHERE id = $3 AND school_id = $4
			`, req.ToClassID, time.Now().UTC(), existingID, schoolID)
		} else if errors.Is(err, sql.ErrNoRows) {
			enrollment := StudentEnrollment{
				BaseModel:  NewBaseModel(),
				SchoolID:   schoolID,
				StudentID:  studentID,
				ClassID:    req.ToClassID,
				TermID:     req.TermID,
				Status:     EnrollmentActive,
				EnrolledAt: time.Now().UTC(),
			}
			err = enrollment.Create(dbx)
		}

		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func GetCurrentEnrollment(dbx DBTX, schoolID, studentID uuid.UUID) (*StudentEnrollment, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	enrollment := StudentEnrollment{}
	err := dbx.GetContext(ctx, &enrollment, `
		SELECT e.id, e.school_id, e.student_id, e.class_id, e.term_id, e.status, e.enrolled_at, e.created_at, e.updated_at
		FROM student_enrollments e
		INNER JOIN terms t ON t.id = e.term_id
		WHERE e.school_id = $1
		  AND e.student_id = $2
		  AND e.status = 'active'
		  AND t.is_current = true
		LIMIT 1
	`, schoolID, studentID)
	if err != nil {
		return nil, err
	}
	return &enrollment, nil
}

func (e *StudentEnrollment) ToResponse() EnrollmentResponse {
	return EnrollmentResponse{
		ID:         e.ID,
		SchoolID:   e.SchoolID,
		StudentID:  e.StudentID,
		ClassID:    e.ClassID,
		TermID:     e.TermID,
		Status:     e.Status,
		EnrolledAt: e.EnrolledAt,
	}
}

type StudentListFilters struct {
	Search  string
	ClassID *uuid.UUID
	Page    int
	Limit   int
}

type StudentResponse struct {
	ID          uuid.UUID  `json:"id"`
	FirstName   string     `json:"first_name"`
	MiddleName  string     `json:"middle_name"`
	LastName    string     `json:"last_name"`
	Gender      Gender     `json:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	NIN         string     `json:"nin"`
	AdmissionNo string     `json:"admission_no,omitempty"`
}

func ListStudents(dbx DBTX, schoolID uuid.UUID, filters StudentListFilters) ([]StudentResponse, int64, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 {
		filters.Limit = 20
	}
	offset := (filters.Page - 1) * filters.Limit

	baseQuery := `
		FROM students s
		INNER JOIN student_admission sa ON sa.student_id = s.id AND sa.deleted_at IS NULL
		LEFT JOIN student_enrollments e ON e.student_id = s.id AND e.status = 'active'
		LEFT JOIN terms t ON t.id = e.term_id AND t.is_current = true
		WHERE sa.school_id = $1 AND s.deleted_at IS NULL
	`
	args := []interface{}{schoolID}
	argIndex := 2

	if filters.Search != "" {
		baseQuery += fmt.Sprintf(` AND (s.first_name ILIKE $%d OR s.last_name ILIKE $%d OR sa.admission_no ILIKE $%d)`, argIndex, argIndex, argIndex)
		args = append(args, "%"+filters.Search+"%")
		argIndex++
	}
	if filters.ClassID != nil {
		baseQuery += fmt.Sprintf(" AND e.class_id = $%d AND t.is_current = true", argIndex)
		args = append(args, *filters.ClassID)
		argIndex++
	}

	var total int64
	countQuery := "SELECT COUNT(DISTINCT s.id) " + baseQuery
	if err := dbx.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	selectQuery := `
		SELECT DISTINCT s.id, s.first_name, s.middle_name, s.last_name, s.gender, s.date_of_birth, s.nin, sa.admission_no
	` + baseQuery + fmt.Sprintf(`
		ORDER BY s.last_name, s.first_name
		LIMIT $%d OFFSET $%d
	`, argIndex, argIndex+1)
	args = append(args, filters.Limit, offset)

	rows, err := dbx.QueryxContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	students := []StudentResponse{}
	for rows.Next() {
		var student StudentResponse
		if err := rows.Scan(
			&student.ID, &student.FirstName, &student.MiddleName, &student.LastName,
			&student.Gender, &student.DateOfBirth, &student.NIN, &student.AdmissionNo,
		); err != nil {
			return nil, 0, err
		}
		students = append(students, student)
	}
	return students, total, nil
}

func ResolveFeeAmount(dbx DBTX, feeTypeID, classID uuid.UUID, defaultAmount decimal.Decimal) (decimal.Decimal, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	var amount decimal.Decimal
	err := dbx.GetContext(ctx, &amount, `
		SELECT amount FROM fee_class_amounts
		WHERE fee_type_id = $1 AND class_id = $2
	`, feeTypeID, classID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultAmount, nil
		}
		return decimal.Zero, err
	}
	return amount, nil
}
