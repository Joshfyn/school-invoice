package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InvoiceStatus represents the status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusPending  InvoiceStatus = "pending"
	InvoiceStatusPartial  InvoiceStatus = "partial"
	InvoiceStatusPaid     InvoiceStatus = "paid"
	InvoiceStatusOverdue  InvoiceStatus = "overdue"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
)

// Invoice represents an invoice for a student
type Invoice struct {
	BaseModel
	SchoolID       uuid.UUID       `json:"school_id"`
	EnrollmentID   uuid.UUID       `json:"enrollment_id"`
	InvoiceNo      string          `json:"invoice_no"`
	TotalAmount    decimal.Decimal `json:"total_amount"`
	AmountPaid     decimal.Decimal `json:"amount_paid"`
	Status         InvoiceStatus   `json:"status"`
	DueDate        time.Time       `json:"due_date"`
	GraceDate      *time.Time      `json:"grace_date,omitempty"`
	GraceGrantedBy *uuid.UUID      `json:"grace_granted_by,omitempty"`
}

// CreateInvoiceRequest is the request body for creating an invoice
type CreateInvoiceRequest struct {
	StudentID   uuid.UUID   `json:"student_id" binding:"required"`
	FeeTypeIDs  []uuid.UUID `json:"fee_type_ids" binding:"required,min=1"`
	DueDate     string      `json:"due_date" binding:"required"` // Format: YYYY-MM-DD
	CustomItems []CustomInvoiceItem `json:"custom_items,omitempty"`
}

// CustomInvoiceItem represents a custom item to add to an invoice
type CustomInvoiceItem struct {
	Description string  `json:"description" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	IsOptional  bool    `json:"is_optional"`
}

// BulkCreateInvoiceRequest is the request body for creating invoices for multiple students
type BulkCreateInvoiceRequest struct {
	ClassID     *uuid.UUID  `json:"class_id,omitempty"`     // Create for all students in class
	StudentIDs  []uuid.UUID `json:"student_ids,omitempty"`  // Or specify specific students
	FeeTypeIDs  []uuid.UUID `json:"fee_type_ids" binding:"required,min=1"`
	DueDate     string      `json:"due_date" binding:"required"`
}

// GrantGraceRequest is the request body for granting a grace period
type GrantGraceRequest struct {
	GraceDate string `json:"grace_date" binding:"required"` // Format: YYYY-MM-DD
	Reason    string `json:"reason"`
}

// InvoiceResponse is the response for invoice data
type InvoiceResponse struct {
	ID             uuid.UUID             `json:"id"`
	SchoolID       uuid.UUID             `json:"school_id"`
	EnrollmentID   uuid.UUID             `json:"enrollment_id"`
	InvoiceNo      string                `json:"invoice_no"`
	TotalAmount    float64               `json:"total_amount"`
	AmountPaid     float64               `json:"amount_paid"`
	AmountDue      float64               `json:"amount_due"`
	Status         InvoiceStatus         `json:"status"`
	DueDate        string                `json:"due_date"`
	GraceDate      *string               `json:"grace_date,omitempty"`
	GraceGrantedBy *uuid.UUID            `json:"grace_granted_by,omitempty"`
	Student        *StudentResponse      `json:"student,omitempty"`
	//Enrollment     *EnrollmentResponse   `json:"enrollment,omitempty"`
	Items          []InvoiceItemResponse `json:"items,omitempty"`
	Payments       []PaymentResponse     `json:"payments,omitempty"`
}

func (i *Invoice) ToResponse() InvoiceResponse {
	totalAmount, _ := i.TotalAmount.Float64()
	amountPaid, _ := i.AmountPaid.Float64()
	
	resp := InvoiceResponse{
		ID:             i.ID,
		SchoolID:       i.SchoolID,
		EnrollmentID:   i.EnrollmentID,
		InvoiceNo:      i.InvoiceNo,
		TotalAmount:    totalAmount,
		AmountPaid:     amountPaid,
		AmountDue:      totalAmount - amountPaid,
		Status:         i.Status,
		DueDate:        i.DueDate.Format("2006-01-02"),
		GraceGrantedBy: i.GraceGrantedBy,
	}
	
	if i.GraceDate != nil {
		graceDate := i.GraceDate.Format("2006-01-02")
		resp.GraceDate = &graceDate
	}
	
	return resp
}

// InvoiceItem represents a line item in an invoice
type InvoiceItem struct {
	BaseModel
	SchoolID    uuid.UUID       `json:"school_id"`
	InvoiceID   uuid.UUID       `json:"invoice_id"`
	FeeTypeID   *uuid.UUID      `json:"fee_type_id,omitempty"` // nil for custom items
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	IsOptional  bool            `json:"is_optional"`
}

// InvoiceItemResponse is the response for invoice item data
type InvoiceItemResponse struct {
	ID          uuid.UUID        `json:"id"`
	InvoiceID   uuid.UUID        `json:"invoice_id"`
	FeeTypeID   *uuid.UUID       `json:"fee_type_id,omitempty"`
	Description string           `json:"description"`
	Amount      float64          `json:"amount"`
	IsOptional  bool             `json:"is_optional"`
	FeeType     *FeeTypeResponse `json:"fee_type,omitempty"`
}

func (ii *InvoiceItem) ToResponse() InvoiceItemResponse {
	amount, _ := ii.Amount.Float64()
	return InvoiceItemResponse{
		ID:          ii.ID,
		InvoiceID:   ii.InvoiceID,
		FeeTypeID:   ii.FeeTypeID,
		Description: ii.Description,
		Amount:      amount,
		IsOptional:  ii.IsOptional,
	}
}

// InvoiceListFilters represents filters for listing invoices
type InvoiceListFilters struct {
	Status  *InvoiceStatus `form:"status"`
	ClassID *uuid.UUID     `form:"class_id"`
	TermID  *uuid.UUID     `form:"term_id"`
	Search  string         `form:"search"` // Search by invoice_no or student name
	Page    int            `form:"page,default=1"`
	Limit   int            `form:"limit,default=20"`
}
