package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrFeeCategoryInUse = errors.New("fee category is in use by fee types")

type FeeCategoryRecord struct {
	BaseModel
	SchoolID uuid.UUID `json:"school_id" db:"school_id"`
	Name     string    `json:"name" db:"name"`
	Code     string    `json:"code" db:"code"`
	IsActive bool      `json:"is_active" db:"is_active"`
}

type CreateFeeCategoryRequest struct {
	Name string `json:"name" binding:"required,min=2"`
	Code string `json:"code" binding:"required,min=2,max=50"`
}

func DefaultFeeCategories() []struct{ Name, Code string } {
	return []struct{ Name, Code string }{
		{"Academic", "academic"},
		{"Uniform", "uniform"},
		{"Materials", "materials"},
		{"Extra Curricular", "extra_curricular"},
		{"Other", "other"},
	}
}

func SeedFeeCategories(dbx DBTX, schoolID uuid.UUID) error {
	for _, cat := range DefaultFeeCategories() {
		fc := FeeCategoryRecord{
			BaseModel: NewBaseModel(),
			SchoolID:  schoolID,
			Name:      cat.Name,
			Code:      cat.Code,
			IsActive:  true,
		}
		if err := fc.CreateIfNotExists(dbx); err != nil {
			return err
		}
	}
	return nil
}

func (fc *FeeCategoryRecord) CreateIfNotExists(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if fc.ID == uuid.Nil {
		fc.ID = uuid.New()
	}

	_, err := dbx.ExecContext(ctx, `
		INSERT INTO fee_categories (id, school_id, name, code, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (school_id, code) DO NOTHING
	`, fc.ID, fc.SchoolID, fc.Name, fc.Code, fc.IsActive)
	return err
}

func (fc *FeeCategoryRecord) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if fc.ID == uuid.Nil {
		fc.ID = uuid.New()
	}

	rows, err := NamedQueryContext(dbx, ctx, `
		INSERT INTO fee_categories (id, school_id, name, code, is_active)
		VALUES (:id, :school_id, :name, :code, :is_active)
		RETURNING id, school_id, name, code, is_active, created_at, updated_at
	`, fc)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("fee category not created")
	}
	return rows.StructScan(fc)
}

func ListFeeCategories(dbx DBTX, schoolID uuid.UUID) ([]FeeCategoryRecord, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	categories := []FeeCategoryRecord{}
	err := dbx.SelectContext(ctx, &categories, `
		SELECT id, school_id, name, code, is_active, created_at, updated_at
		FROM fee_categories
		WHERE school_id = $1 AND is_active = true
		ORDER BY name
	`, schoolID)
	return categories, err
}

func FeeCategoryExists(dbx DBTX, schoolID uuid.UUID, code string) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	var count int
	err := dbx.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM fee_categories
		WHERE school_id = $1 AND code = $2 AND is_active = true
	`, schoolID, code)
	return count > 0, err
}

func (fc *FeeCategoryRecord) Delete(dbx DBTX, schoolID uuid.UUID) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	var count int
	if err := dbx.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM fee_types WHERE school_id = $1 AND category = $2
	`, schoolID, fc.Code); err != nil {
		return err
	}
	if count > 0 {
		return ErrFeeCategoryInUse
	}

	_, err := dbx.ExecContext(ctx, `
		UPDATE fee_categories SET is_active = false, updated_at = $1
		WHERE id = $2 AND school_id = $3
	`, time.Now().UTC(), fc.ID, schoolID)
	return err
}
