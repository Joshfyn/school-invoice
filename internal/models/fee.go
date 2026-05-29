package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FeeCategory represents the category of a fee type
type FeeCategory string

const (
	FeeCategoryAcademic        FeeCategory = "academic"
	FeeCategoryUniform         FeeCategory = "uniform"
	FeeCategoryMaterials       FeeCategory = "materials"
	FeeCategoryExtraCurricular FeeCategory = "extra_curricular"
	FeeCategoryOther           FeeCategory = "other"
)

// FeeFrequency represents how often a fee applies
type FeeFrequency string

const (
	FeeFrequencyPerTerm    FeeFrequency = "per_term"
	FeeFrequencyPerSession FeeFrequency = "per_session"
	FeeFrequencyOneTime    FeeFrequency = "one_time"
)

// FeeType represents a type of fee that a school charges
type FeeType struct {
	BaseModel
	SchoolID      uuid.UUID       `json:"school_id" db:"school_id"`
	Name          string          `json:"name" db:"name"`
	Description   string          `json:"description" db:"description"`
	DefaultAmount decimal.Decimal `json:"default_amount" db:"default_amount"`
	Category      FeeCategory     `json:"category" db:"category"`
	Frequency     FeeFrequency    `json:"frequency" db:"frequency"`
	IsOptional    bool            `json:"is_optional" db:"is_optional"`
	IsActive      bool            `json:"is_active" db:"is_active"`
}

// CreateFeeTypeRequest is the request body for creating a fee type
type CreateFeeTypeRequest struct {
	Name          string       `json:"name" binding:"required,min=2"`
	Description   string       `json:"description"`
	DefaultAmount float64      `json:"default_amount" binding:"required,gt=0"`
	Category      FeeCategory  `json:"category" binding:"required,oneof=academic uniform materials extra_curricular other"`
	Frequency     FeeFrequency `json:"frequency" binding:"required,oneof=per_term per_session one_time"`
	IsOptional    bool         `json:"is_optional"`
}

// UpdateFeeTypeRequest is the request body for updating a fee type
type UpdateFeeTypeRequest struct {
	Name          *string       `json:"name,omitempty" binding:"omitempty,min=2"`
	Description   *string       `json:"description,omitempty"`
	DefaultAmount *float64      `json:"default_amount,omitempty" binding:"omitempty,gt=0"`
	Category      *FeeCategory  `json:"category,omitempty" binding:"omitempty,oneof=academic uniform materials extra_curricular other"`
	Frequency     *FeeFrequency `json:"frequency,omitempty" binding:"omitempty,oneof=per_term per_session one_time"`
	IsOptional    *bool         `json:"is_optional,omitempty"`
	IsActive      *bool         `json:"is_active,omitempty"`
}

// FeeTypeResponse is the response for fee type data
type FeeTypeResponse struct {
	ID            uuid.UUID    `json:"id"`
	SchoolID      uuid.UUID    `json:"school_id"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	DefaultAmount float64      `json:"default_amount"`
	Category      FeeCategory  `json:"category"`
	Frequency     FeeFrequency `json:"frequency"`
	IsOptional    bool         `json:"is_optional"`
	IsActive      bool         `json:"is_active"`
}

func (f *FeeType) ToResponse() FeeTypeResponse {
	amount, _ := f.DefaultAmount.Float64()
	return FeeTypeResponse{
		ID:            f.ID,
		SchoolID:      f.SchoolID,
		Name:          f.Name,
		Description:   f.Description,
		DefaultAmount: amount,
		Category:      f.Category,
		Frequency:     f.Frequency,
		IsOptional:    f.IsOptional,
		IsActive:      f.IsActive,
	}
}

// FeeClassAmount represents a class-specific fee amount
type FeeClassAmount struct {
	BaseModel
	FeeTypeID uuid.UUID       `json:"fee_type_id" db:"fee_type_id"`
	ClassID   uuid.UUID       `json:"class_id" db:"class_id"`
	Amount    decimal.Decimal `json:"amount" db:"amount"`
}

// SetFeeClassAmountsRequest is the request body for setting class-specific amounts
type SetFeeClassAmountsRequest struct {
	Amounts []FeeClassAmountItem `json:"amounts" binding:"required,dive"`
}

// FeeClassAmountItem represents a single class-amount pair
type FeeClassAmountItem struct {
	ClassID uuid.UUID `json:"class_id" binding:"required"`
	Amount  float64   `json:"amount" binding:"required,gte=0"`
}

// FeeClassAmountResponse is the response for fee class amount data
type FeeClassAmountResponse struct {
	FeeTypeID uuid.UUID      `json:"fee_type_id"`
	ClassID   uuid.UUID      `json:"class_id"`
	Amount    float64        `json:"amount"`
	Class     *ClassResponse `json:"class,omitempty"`
}

func (f *FeeClassAmount) ToResponse() FeeClassAmountResponse {
	amount, _ := f.Amount.Float64()
	return FeeClassAmountResponse{
		FeeTypeID: f.FeeTypeID,
		ClassID:   f.ClassID,
		Amount:    amount,
	}
}

func (f *FeeType) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	if !f.IsActive {
		f.IsActive = true
	}

	rows, err := NamedQueryContext(dbx, ctx, `
		INSERT INTO fee_types (id, school_id, name, description, default_amount, category, frequency, is_optional, is_active)
		VALUES (:id, :school_id, :name, :description, :default_amount, :category, :frequency, :is_optional, :is_active)
		RETURNING id, school_id, name, description, default_amount, category, frequency, is_optional, is_active, created_at, updated_at
	`, f)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("fee type not created")
	}
	return rows.StructScan(f)
}

func (f *FeeType) Get(dbx DBTX, schoolID uuid.UUID) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	return dbx.GetContext(ctx, f, `
		SELECT id, school_id, name, description, default_amount, category, frequency, is_optional, is_active, created_at, updated_at
		FROM fee_types
		WHERE id = $1 AND school_id = $2
	`, f.ID, schoolID)
}

func (f *FeeType) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		UPDATE fee_types
		SET name = $1, description = $2, default_amount = $3, category = $4, frequency = $5,
		    is_optional = $6, is_active = $7, updated_at = $8
		WHERE id = $9 AND school_id = $10
	`, f.Name, f.Description, f.DefaultAmount, f.Category, f.Frequency, f.IsOptional, f.IsActive, time.Now().UTC(), f.ID, f.SchoolID)
	return err
}

func ListFeeTypes(dbx DBTX, schoolID uuid.UUID, activeOnly bool) ([]FeeType, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	query := `
		SELECT id, school_id, name, description, default_amount, category, frequency, is_optional, is_active, created_at, updated_at
		FROM fee_types
		WHERE school_id = $1
	`
	if activeOnly {
		query += " AND is_active = true"
	}
	query += " ORDER BY name"

	feeTypes := []FeeType{}
	if err := dbx.SelectContext(ctx, &feeTypes, query, schoolID); err != nil {
		return nil, err
	}
	return feeTypes, nil
}

func ListFeeClassAmounts(dbx DBTX, feeTypeID uuid.UUID) ([]FeeClassAmount, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	amounts := []FeeClassAmount{}
	err := dbx.SelectContext(ctx, &amounts, `
		SELECT id, fee_type_id, class_id, amount, created_at, updated_at
		FROM fee_class_amounts
		WHERE fee_type_id = $1
	`, feeTypeID)
	return amounts, err
}

func SetFeeClassAmounts(dbx DBTX, feeTypeID uuid.UUID, items []FeeClassAmountItem) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if _, err := dbx.ExecContext(ctx, `DELETE FROM fee_class_amounts WHERE fee_type_id = $1`, feeTypeID); err != nil {
		return err
	}

	for _, item := range items {
		amount := FeeClassAmount{
			BaseModel: NewBaseModel(),
			FeeTypeID: feeTypeID,
			ClassID:   item.ClassID,
			Amount:    decimal.NewFromFloat(item.Amount),
		}
		if _, err := dbx.ExecContext(ctx, `
			INSERT INTO fee_class_amounts (id, fee_type_id, class_id, amount)
			VALUES ($1, $2, $3, $4)
		`, amount.ID, amount.FeeTypeID, amount.ClassID, amount.Amount); err != nil {
			return err
		}
	}
	return nil
}
