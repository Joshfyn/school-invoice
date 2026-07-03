package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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

// SendInvoiceRequest is the optional body for sending an invoice PDF by email.
type SendInvoiceRequest struct {
	Email string `json:"email,omitempty" binding:"omitempty,email"`
}

// UpdateInvoiceStatusRequest is the request body for changing invoice status (super admin).
type UpdateInvoiceStatusRequest struct {
	Status InvoiceStatus `json:"status" binding:"required,oneof=pending partial paid overdue cancelled"`
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
	CreatedAt      string                `json:"created_at,omitempty"`
	StudentName    string                `json:"student_name,omitempty"`
	AdmissionNo    string                `json:"admission_no,omitempty"`
	ClassName      string                `json:"class_name,omitempty"`
	GuardianName   string                `json:"guardian_name,omitempty"`
	GuardianEmail  string                `json:"guardian_email,omitempty"`
	LastSentAt     *string               `json:"last_sent_at,omitempty"`
	SendHistory    []InvoiceSendLogResponse `json:"send_history,omitempty"`
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
	if !i.CreatedAt.IsZero() {
		resp.CreatedAt = i.CreatedAt.Format(time.RFC3339)
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

var ErrInvoiceHasPayments = errors.New("invoice has payments and cannot be deleted")

func (i *Invoice) UpdateStatus(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		UPDATE invoices SET status = $1, updated_at = $2
		WHERE id = $3 AND school_id = $4
	`, i.Status, time.Now().UTC(), i.ID, i.SchoolID)
	return err
}

func (i *Invoice) Delete(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	var paymentCount int
	if err := dbx.GetContext(ctx, &paymentCount, `
		SELECT COUNT(*) FROM payments WHERE invoice_id = $1
	`, i.ID); err != nil {
		return err
	}
	if paymentCount > 0 {
		return ErrInvoiceHasPayments
	}

	_, err := dbx.ExecContext(ctx, `DELETE FROM invoices WHERE id = $1 AND school_id = $2`, i.ID, i.SchoolID)
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

// InvoicePDFContext holds all data needed to render and email an invoice PDF.
type InvoicePDFContext struct {
	Invoice       Invoice
	Items         []InvoiceItem
	School        School
	StudentName   string
	AdmissionNo   string
	ClassName     string
	GuardianName  string
	GuardianEmail string
	GuardianAddr  string
}

func GetInvoicePDFContext(dbx DBTX, schoolID, invoiceID uuid.UUID) (*InvoicePDFContext, error) {
	invoice := Invoice{BaseModel: BaseModel{ID: invoiceID}}
	if err := invoice.Get(dbx, schoolID); err != nil {
		return nil, err
	}

	items, err := ListInvoiceItems(dbx, invoiceID)
	if err != nil {
		return nil, err
	}

	school := School{BaseModel: BaseModel{ID: schoolID}}
	if err := school.GetProfile(dbx); err != nil {
		return nil, err
	}

	enrollment := StudentEnrollment{BaseModel: BaseModel{ID: invoice.EnrollmentID}}
	if err := enrollment.Get(dbx, schoolID); err != nil {
		return nil, err
	}

	student := Student{ID: enrollment.StudentID}
	if err := student.Get(dbx); err != nil {
		return nil, err
	}

	class := Class{BaseModel: BaseModel{ID: enrollment.ClassID}}
	className := ""
	if err := class.Get(dbx, schoolID); err == nil {
		className = class.Name + " " + class.Section
	}

	admissionNo := ""
	var admNo string
	ctx, cancel := GetDBContext(dbx)
	defer cancel()
	_ = dbx.GetContext(ctx, &admNo, `
		SELECT admission_no FROM student_admission
		WHERE student_id = $1 AND school_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`, enrollment.StudentID, schoolID)
	admissionNo = admNo

	guardians, err := ListStudentGuardians(dbx, enrollment.StudentID)
	if err != nil {
		return nil, err
	}

	guardianName, guardianEmail, guardianAddr := "", "", ""
	for _, g := range guardians {
		if g.Guardian == nil {
			continue
		}
		if g.IsPrimary || g.ReceivesNotifications || guardianEmail == "" {
			guardianName = g.Guardian.FirstName + " " + g.Guardian.LastName
			guardianEmail = g.Guardian.Email
			guardianAddr = g.Guardian.Address
		}
		if g.IsPrimary && g.Guardian.Email != "" {
			break
		}
	}

	return &InvoicePDFContext{
		Invoice:       invoice,
		Items:         items,
		School:        school,
		StudentName:   student.FirstName + " " + student.LastName,
		AdmissionNo:   admissionNo,
		ClassName:     className,
		GuardianName:  guardianName,
		GuardianEmail: guardianEmail,
		GuardianAddr:  guardianAddr,
	}, nil
}

func GetInvoiceDetail(dbx DBTX, schoolID, invoiceID uuid.UUID) (*InvoiceResponse, error) {
	ctx, err := GetInvoicePDFContext(dbx, schoolID, invoiceID)
	if err != nil {
		return nil, err
	}

	resp := ctx.Invoice.ToResponse()
	resp.StudentName = ctx.StudentName
	resp.AdmissionNo = ctx.AdmissionNo
	resp.ClassName = ctx.ClassName
	resp.GuardianName = ctx.GuardianName
	resp.GuardianEmail = ctx.GuardianEmail

	for _, item := range ctx.Items {
		resp.Items = append(resp.Items, item.ToResponse())
	}

	history, err := ListInvoiceSendLogs(dbx, invoiceID)
	if err != nil {
		return nil, err
	}
	resp.SendHistory = history
	if len(history) > 0 {
		last := history[0].CreatedAt
		resp.LastSentAt = &last
	}

	return &resp, nil
}

func GetLastSentAtByInvoices(dbx DBTX, invoiceIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	if len(invoiceIDs) == 0 {
		return map[uuid.UUID]time.Time{}, nil
	}
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	query, args, err := sqlx.In(`
		SELECT invoice_id, MAX(created_at) AS last_sent_at
		FROM invoice_send_logs
		WHERE invoice_id IN (?)
		GROUP BY invoice_id
	`, invoiceIDs)
	if err != nil {
		return nil, err
	}
	query = dbx.Rebind(query)

	rows := []struct {
		InvoiceID  uuid.UUID `db:"invoice_id"`
		LastSentAt time.Time `db:"last_sent_at"`
	}{}
	if err := dbx.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]time.Time, len(rows))
	for _, row := range rows {
		result[row.InvoiceID] = row.LastSentAt
	}
	return result, nil
}

func (i *Invoice) GrantGrace(dbx DBTX, userID uuid.UUID, graceDate time.Time) error {
	i.GraceDate = &graceDate
	i.GraceGrantedBy = &userID
	return i.Update(dbx)
}
