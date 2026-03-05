package models

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
	HasMore    bool        `json:"has_more"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, page, limit int, totalItems int64) PaginatedResponse {
	totalPages := int(totalItems) / limit
	if int(totalItems)%limit > 0 {
		totalPages++
	}

	return PaginatedResponse{
		Data:       data,
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string            `json:"error,omitempty"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Redis    string `json:"redis"`
	Version  string `json:"version"`
}

// DashboardSummary represents the dashboard summary statistics
type DashboardSummary struct {
	TotalStudents    int64             `json:"total_students"`
	TotalInvoices    int64             `json:"total_invoices"`
	PendingInvoices  int64             `json:"pending_invoices"`
	OverdueInvoices  int64             `json:"overdue_invoices"`
	TotalCollected   float64           `json:"total_collected"`
	TotalOutstanding float64           `json:"total_outstanding"`
	CollectionRate   float64           `json:"collection_rate"` // percentage
	RecentPayments   []PaymentResponse `json:"recent_payments,omitempty"`
}

// OutstandingReport represents outstanding fees report
type OutstandingReport struct {
	StudentID    string  `json:"student_id"`
	StudentName  string  `json:"student_name"`
	AdmissionNo  string  `json:"admission_no"`
	ClassName    string  `json:"class_name"`
	TotalDue     float64 `json:"total_due"`
	AmountPaid   float64 `json:"amount_paid"`
	Outstanding  float64 `json:"outstanding"`
	InvoiceCount int     `json:"invoice_count"`
}

// CollectionReport represents collection report data
type CollectionReport struct {
	Period       string             `json:"period"` // e.g., "2024-01", "2024-01-15"
	TotalAmount  float64            `json:"total_amount"`
	PaymentCount int                `json:"payment_count"`
	ByMethod     map[string]float64 `json:"by_method"`
}
