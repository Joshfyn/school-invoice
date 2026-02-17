package models

import (
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
	SchoolID      uuid.UUID       `json:"school_id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	DefaultAmount decimal.Decimal `json:"default_amount"`
	Category      FeeCategory     `json:"category"`
	Frequency     FeeFrequency    `json:"frequency"`
	IsOptional    bool            `json:"is_optional"`
	IsActive      bool            `json:"is_active"`
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
	FeeTypeID uuid.UUID       `json:"fee_type_id"`
	ClassID   uuid.UUID       `json:"class_id"`
	Amount    decimal.Decimal `json:"amount"`
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
