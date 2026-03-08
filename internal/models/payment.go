package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusSuccess  PaymentStatus = "success"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// PaymentMethod represents the method of payment
type PaymentMethod string

const (
	PaymentMethodCard PaymentMethod = "card"
	PaymentMethodBank PaymentMethod = "bank_transfer"
	PaymentMethodUSSD PaymentMethod = "ussd"
	PaymentMethodCash PaymentMethod = "cash"
	PaymentMethodPOS  PaymentMethod = "pos"
)

// Payment represents a payment made towards an invoice
type Payment struct {
	BaseModel
	SchoolID      uuid.UUID              `json:"school_id"`
	InvoiceID     uuid.UUID              `json:"invoice_id"`
	Amount        decimal.Decimal        `json:"amount"`
	PaymentMethod PaymentMethod          `json:"payment_method"`
	Reference     string                 `json:"reference"` // Payment gateway reference
	BankName      string                 `json:"bank_name"` // For bank transfers
	Status        PaymentStatus          `json:"status"`
	PaidAt        *time.Time             `json:"paid_at,omitempty"`
	VerifiedBy    *uuid.UUID             `json:"verified_by,omitempty"` // For manual bank verification
	Metadata      map[string]interface{} `json:"metadata,omitempty"`    // Additional payment info
}

// InitializePaymentRequest is the request body for initializing a payment
type InitializePaymentRequest struct {
	InvoiceID uuid.UUID `json:"invoice_id" binding:"required"`
	Amount    float64   `json:"amount" binding:"required,gt=0"`
	Email     string    `json:"email" binding:"required,email"`
	Phone     string    `json:"phone"`
}

// InitializePaymentResponse is the response for payment initialization
type InitializePaymentResponse struct {
	Reference        string `json:"reference"`
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
}

// VerifyBankPaymentRequest is the request body for manually verifying a bank payment
type VerifyBankPaymentRequest struct {
	InvoiceID   uuid.UUID `json:"invoice_id" binding:"required"`
	Amount      float64   `json:"amount" binding:"required,gt=0"`
	BankName    string    `json:"bank_name" binding:"required"`
	Reference   string    `json:"reference"`                       // Bank transfer reference/narration
	PaymentDate string    `json:"payment_date" binding:"required"` // Format: YYYY-MM-DD
	Notes       string    `json:"notes"`
}

// PaymentResponse is the response for payment data
type PaymentResponse struct {
	ID            uuid.UUID        `json:"id"`
	SchoolID      uuid.UUID        `json:"school_id"`
	InvoiceID     uuid.UUID        `json:"invoice_id"`
	Amount        float64          `json:"amount"`
	PaymentMethod PaymentMethod    `json:"payment_method"`
	Reference     string           `json:"reference"`
	BankName      string           `json:"bank_name,omitempty"`
	Status        PaymentStatus    `json:"status"`
	PaidAt        *string          `json:"paid_at,omitempty"`
	VerifiedBy    *uuid.UUID       `json:"verified_by,omitempty"`
	Invoice       *InvoiceResponse `json:"invoice,omitempty"`
}

func (p *Payment) ToResponse() PaymentResponse {
	amount, _ := p.Amount.Float64()

	resp := PaymentResponse{
		ID:            p.ID,
		SchoolID:      p.SchoolID,
		InvoiceID:     p.InvoiceID,
		Amount:        amount,
		PaymentMethod: p.PaymentMethod,
		Reference:     p.Reference,
		BankName:      p.BankName,
		Status:        p.Status,
		VerifiedBy:    p.VerifiedBy,
	}

	if p.PaidAt != nil {
		paidAt := p.PaidAt.Format(time.RFC3339)
		resp.PaidAt = &paidAt
	}

	return resp
}

// PaystackWebhookPayload represents the webhook payload from Paystack
type PaystackWebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		ID              int    `json:"id"`
		Reference       string `json:"reference"`
		Amount          int    `json:"amount"` // Amount in kobo
		Currency        string `json:"currency"`
		Status          string `json:"status"`
		GatewayResponse string `json:"gateway_response"`
		PaidAt          string `json:"paid_at"`
		Channel         string `json:"channel"`
		Customer        struct {
			Email string `json:"email"`
			Phone string `json:"phone"`
		} `json:"customer"`
		Metadata map[string]interface{} `json:"metadata"`
	} `json:"data"`
}

// ReceiptData represents data for generating a receipt
type ReceiptData struct {
	Payment     PaymentResponse    `json:"payment"`
	Invoice     InvoiceResponse    `json:"invoice"`
	//Student     StudentResponse    `json:"student"`
	//School      dto.SchoolResponse `json:"school"`
	ReceiptNo   string             `json:"receipt_no"`
	GeneratedAt string             `json:"generated_at"`
}
