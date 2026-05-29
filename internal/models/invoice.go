package models

import (
	"errors"
	"fmt"
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
	SchoolID       uuid.UUID       `json:"school_id" db:"school_id"`
	EnrollmentID   uuid.UUID       `json:"enrollment_id" db:"enrollment_id"`
	InvoiceNo      string          `json:"invoice_no" db:"invoice_no"`
	TotalAmount    decimal.Decimal `json:"total_amount" db:"total_amount"`
	AmountPaid     decimal.Decimal `json:"amount_paid" db:"amount_paid"`
	Status         InvoiceStatus   `json:"status" db:"status"`
	DueDate        time.Time       `json:"due_date" db:"due_date"`
	GraceDate      *time.Time      `json:"grace_date,omitempty" db:"grace_date"`
	GraceGrantedBy *uuid.UUID      `json:"grace_granted_by,omitempty" db:"grace_granted_by"`
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
	//Student        *StudentResponse      `json:"student,omitempty"`
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
	SchoolID    uuid.UUID       `json:"school_id" db:"school_id"`
	InvoiceID   uuid.UUID       `json:"invoice_id" db:"invoice_id"`
	FeeTypeID   *uuid.UUID      `json:"fee_type_id,omitempty" db:"fee_type_id"`
	Description string          `json:"description" db:"description"`
	Amount      decimal.Decimal `json:"amount" db:"amount"`
	IsOptional  bool            `json:"is_optional" db:"is_optional"`
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

func (i *Invoice) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.Status == "" {
		i.Status = InvoiceStatusPending
	}

	rows, err := NamedQueryContext(dbx, ctx, `
		INSERT INTO invoices (id, school_id, enrollment_id, invoice_no, total_amount, amount_paid, status, due_date, grace_date, grace_granted_by)
		VALUES (:id, :school_id, :enrollment_id, :invoice_no, :total_amount, :amount_paid, :status, :due_date, :grace_date, :grace_granted_by)
		RETURNING id, school_id, enrollment_id, invoice_no, total_amount, amount_paid, status, due_date, grace_date, grace_granted_by, created_at, updated_at
	`, i)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("invoice not created")
	}
	return rows.StructScan(i)
}

func (i *Invoice) Get(dbx DBTX, schoolID uuid.UUID) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	return dbx.GetContext(ctx, i, `
		SELECT id, school_id, enrollment_id, invoice_no, total_amount, amount_paid, status, due_date, grace_date, grace_granted_by, created_at, updated_at
		FROM invoices
		WHERE id = $1 AND school_id = $2
	`, i.ID, schoolID)
}

func (i *Invoice) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		UPDATE invoices
		SET total_amount = $1, amount_paid = $2, status = $3, due_date = $4,
		    grace_date = $5, grace_granted_by = $6, updated_at = $7
		WHERE id = $8 AND school_id = $9
	`, i.TotalAmount, i.AmountPaid, i.Status, i.DueDate, i.GraceDate, i.GraceGrantedBy, time.Now().UTC(), i.ID, i.SchoolID)
	return err
}

func (ii *InvoiceItem) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if ii.ID == uuid.Nil {
		ii.ID = uuid.New()
	}

	rows, err := NamedQueryContext(dbx, ctx, `
		INSERT INTO invoice_items (id, school_id, invoice_id, fee_type_id, description, amount, is_optional)
		VALUES (:id, :school_id, :invoice_id, :fee_type_id, :description, :amount, :is_optional)
		RETURNING id, school_id, invoice_id, fee_type_id, description, amount, is_optional, created_at, updated_at
	`, ii)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("invoice item not created")
	}
	return rows.StructScan(ii)
}

func ListInvoiceItems(dbx DBTX, invoiceID uuid.UUID) ([]InvoiceItem, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	items := []InvoiceItem{}
	err := dbx.SelectContext(ctx, &items, `
		SELECT id, school_id, invoice_id, fee_type_id, description, amount, is_optional, created_at, updated_at
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY description
	`, invoiceID)
	return items, err
}

func ListInvoices(dbx DBTX, schoolID uuid.UUID, filters InvoiceListFilters) ([]Invoice, int64, error) {
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
		FROM invoices i
		INNER JOIN student_enrollments e ON e.id = i.enrollment_id
		WHERE i.school_id = $1
	`
	args := []interface{}{schoolID}
	argIndex := 2

	if filters.Status != nil {
		baseQuery += fmt.Sprintf(" AND i.status = $%d", argIndex)
		args = append(args, *filters.Status)
		argIndex++
	}
	if filters.ClassID != nil {
		baseQuery += fmt.Sprintf(" AND e.class_id = $%d", argIndex)
		args = append(args, *filters.ClassID)
		argIndex++
	}
	if filters.TermID != nil {
		baseQuery += fmt.Sprintf(" AND e.term_id = $%d", argIndex)
		args = append(args, *filters.TermID)
		argIndex++
	}
	if filters.Search != "" {
		baseQuery += fmt.Sprintf(" AND i.invoice_no ILIKE $%d", argIndex)
		args = append(args, "%"+filters.Search+"%")
		argIndex++
	}

	var total int64
	if err := dbx.GetContext(ctx, &total, "SELECT COUNT(*) "+baseQuery, args...); err != nil {
		return nil, 0, err
	}

	selectQuery := `
		SELECT i.id, i.school_id, i.enrollment_id, i.invoice_no, i.total_amount, i.amount_paid,
		       i.status, i.due_date, i.grace_date, i.grace_granted_by, i.created_at, i.updated_at
	` + baseQuery + fmt.Sprintf(`
		ORDER BY i.created_at DESC
		LIMIT $%d OFFSET $%d
	`, argIndex, argIndex+1)
	args = append(args, filters.Limit, offset)

	invoices := []Invoice{}
	if err := dbx.SelectContext(ctx, &invoices, selectQuery, args...); err != nil {
		return nil, 0, err
	}
	return invoices, total, nil
}

func ListGuardianInvoices(dbx DBTX, schoolID, guardianID uuid.UUID) ([]Invoice, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	invoices := []Invoice{}
	err := dbx.SelectContext(ctx, &invoices, `
		SELECT DISTINCT i.id, i.school_id, i.enrollment_id, i.invoice_no, i.total_amount, i.amount_paid,
		       i.status, i.due_date, i.grace_date, i.grace_granted_by, i.created_at, i.updated_at
		FROM invoices i
		INNER JOIN student_enrollments e ON e.id = i.enrollment_id
		INNER JOIN student_guardians sg ON sg.student_id = e.student_id
		WHERE i.school_id = $1 AND sg.guardian_id = $2
		ORDER BY i.created_at DESC
	`, schoolID, guardianID)
	return invoices, err
}

func GenerateInvoiceNo(schoolID uuid.UUID) string {
	return fmt.Sprintf("INV-%s-%d", schoolID.String()[:8], time.Now().UnixNano()%1000000)
}

func BuildInvoiceFromRequest(dbx DBTX, schoolID uuid.UUID, req CreateInvoiceRequest) (*Invoice, []InvoiceItem, error) {
	enrollment, err := GetCurrentEnrollment(dbx, schoolID, req.StudentID)
	if err != nil {
		return nil, nil, err
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid due date")
	}

	total := decimal.Zero
	items := []InvoiceItem{}

	for _, feeTypeID := range req.FeeTypeIDs {
		feeType := FeeType{BaseModel: BaseModel{ID: feeTypeID}}
		if err := feeType.Get(dbx, schoolID); err != nil {
			return nil, nil, err
		}
		if !feeType.IsActive {
			continue
		}

		amount, err := ResolveFeeAmount(dbx, feeTypeID, enrollment.ClassID, feeType.DefaultAmount)
		if err != nil {
			return nil, nil, err
		}

		feeTypeIDCopy := feeTypeID
		items = append(items, InvoiceItem{
			BaseModel:   NewBaseModel(),
			SchoolID:    schoolID,
			Description: feeType.Name,
			Amount:      amount,
			IsOptional:  feeType.IsOptional,
			FeeTypeID:   &feeTypeIDCopy,
		})
		total = total.Add(amount)
	}

	for _, custom := range req.CustomItems {
		items = append(items, InvoiceItem{
			BaseModel:   NewBaseModel(),
			SchoolID:    schoolID,
			Description: custom.Description,
			Amount:      decimal.NewFromFloat(custom.Amount),
			IsOptional:  custom.IsOptional,
		})
		total = total.Add(decimal.NewFromFloat(custom.Amount))
	}

	invoice := &Invoice{
		BaseModel:    NewBaseModel(),
		SchoolID:     schoolID,
		EnrollmentID: enrollment.ID,
		InvoiceNo:    GenerateInvoiceNo(schoolID),
		TotalAmount:  total,
		AmountPaid:   decimal.Zero,
		Status:       InvoiceStatusPending,
		DueDate:      dueDate,
	}

	return invoice, items, nil
}

func CreateInvoiceWithItems(dbx DBTX, invoice *Invoice, items []InvoiceItem) error {
	if err := invoice.Create(dbx); err != nil {
		return err
	}
	for i := range items {
		items[i].InvoiceID = invoice.ID
		if err := items[i].Create(dbx); err != nil {
			return err
		}
	}
	return nil
}

func GetInvoiceWithItems(dbx DBTX, schoolID, invoiceID uuid.UUID) (*InvoiceResponse, error) {
	invoice := Invoice{BaseModel: BaseModel{ID: invoiceID}}
	if err := invoice.Get(dbx, schoolID); err != nil {
		return nil, err
	}

	resp := invoice.ToResponse()
	items, err := ListInvoiceItems(dbx, invoiceID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		resp.Items = append(resp.Items, item.ToResponse())
	}
	return &resp, nil
}

func (i *Invoice) GrantGrace(dbx DBTX, userID uuid.UUID, graceDate time.Time) error {
	i.GraceDate = &graceDate
	i.GraceGrantedBy = &userID
	return i.Update(dbx)
}
