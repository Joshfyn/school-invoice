package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/models"
	"github.com/school-invoice/backend/lib/mail"
	pdflib "github.com/school-invoice/backend/lib/pdf"
)

type portalGuardian struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
}

type portalSchool struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Phone             string    `json:"phone,omitempty"`
	Email             string    `json:"email,omitempty"`
	Address           string    `json:"address,omitempty"`
	LogoURL           string    `json:"logo_url,omitempty"`
	BankName          string    `json:"bank_name,omitempty"`
	BankAccountName   string    `json:"bank_account_name,omitempty"`
	BankAccountNumber string    `json:"bank_account_number,omitempty"`
}

type portalStudent struct {
	ID          uuid.UUID `json:"id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	AdmissionNo string    `json:"admission_no,omitempty"`
	ClassName   string    `json:"class_name,omitempty"`
}

type portalInvoiceItem struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	IsOptional  bool      `json:"is_optional"`
}

type portalInvoice struct {
	ID          uuid.UUID            `json:"id"`
	InvoiceNo   string               `json:"invoice_no"`
	StudentName string               `json:"student_name"`
	AdmissionNo string               `json:"admission_no,omitempty"`
	ClassName   string               `json:"class_name,omitempty"`
	TotalAmount float64              `json:"total_amount"`
	AmountPaid  float64              `json:"amount_paid"`
	AmountDue   float64              `json:"amount_due"`
	Status      models.InvoiceStatus `json:"status"`
	DueDate     string               `json:"due_date"`
	GraceDate   *string              `json:"grace_date,omitempty"`
	CreatedAt   string               `json:"created_at,omitempty"`
	SchoolName  string               `json:"school_name,omitempty"`
	Items       []portalInvoiceItem  `json:"items"`
}

type portalSessionResponse struct {
	Token     string          `json:"token"`
	ExpiresAt string          `json:"expires_at,omitempty"`
	Guardian  portalGuardian  `json:"guardian"`
	School    portalSchool    `json:"school"`
	Students  []portalStudent `json:"students"`
	Invoices  []portalInvoice `json:"invoices"`
}

// PortalGraceRequest is the request body a guardian submits to ask for a grace-period meeting.
type PortalGraceRequest struct {
	PreferredDate string `json:"preferred_date" binding:"required"`
	PreferredTime string `json:"preferred_time" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
	ContactPhone  string `json:"contact_phone"`
}

// PortalPaymentRequest is the request body for starting an online payment from the portal.
type PortalPaymentRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Email  string  `json:"email"`
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// authorizePortalRequest validates the link token and responds with an error when it is unusable.
func (h *Handler) authorizePortalRequest(c *gin.Context) (*guardianPortalClaims, bool) {
	claims, err := h.parseGuardianPortalToken(c.Param("token"))
	if err != nil {
		h.logger.WithError(err).Warn("Rejected guardian portal token")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_portal_link",
			Message: "This link is invalid or has expired. Please use the link in your most recent invoice email.",
		})
		return nil, false
	}
	return claims, true
}

// loadPortalInvoice loads an invoice only when it belongs to the guardian in the token.
func (h *Handler) loadPortalInvoice(claims *guardianPortalClaims, invoiceID uuid.UUID) (*models.InvoiceResponse, error) {
	invoices, err := models.ListGuardianInvoices(h.dbx, claims.SchoolID, claims.GuardianID)
	if err != nil {
		return nil, err
	}

	for _, invoice := range invoices {
		if invoice.ID == invoiceID {
			return models.GetInvoiceDetail(h.dbx, claims.SchoolID, invoiceID)
		}
	}
	return nil, sql.ErrNoRows
}

func toPortalInvoice(detail *models.InvoiceResponse, schoolName string) portalInvoice {
	items := make([]portalInvoiceItem, 0, len(detail.Items))
	for _, item := range detail.Items {
		items = append(items, portalInvoiceItem{
			ID:          item.ID,
			Description: item.Description,
			Amount:      item.Amount,
			IsOptional:  item.IsOptional,
		})
	}

	return portalInvoice{
		ID:          detail.ID,
		InvoiceNo:   detail.InvoiceNo,
		StudentName: detail.StudentName,
		AdmissionNo: detail.AdmissionNo,
		ClassName:   detail.ClassName,
		TotalAmount: detail.TotalAmount,
		AmountPaid:  detail.AmountPaid,
		AmountDue:   detail.AmountDue,
		Status:      detail.Status,
		DueDate:     detail.DueDate,
		GraceDate:   detail.GraceDate,
		CreatedAt:   detail.CreatedAt,
		SchoolName:  schoolName,
		Items:       items,
	}
}

// GetPortalSession returns everything the guardian portal needs for a link token.
func (h *Handler) GetPortalSession(c *gin.Context) {
	claims, ok := h.authorizePortalRequest(c)
	if !ok {
		return
	}

	school := models.School{BaseModel: models.BaseModel{ID: claims.SchoolID}}
	if err := school.GetProfile(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to load school for guardian portal")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load school details",
		})
		return
	}

	guardianDetail, err := models.GetGuardianWithStudents(h.dbx, claims.SchoolID, claims.GuardianID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "guardian_not_found",
				Message: "We could not find your guardian record for this school.",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to load guardian for portal")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load your details",
		})
		return
	}

	invoices, err := models.ListGuardianInvoices(h.dbx, claims.SchoolID, claims.GuardianID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list guardian invoices for portal")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load your invoices",
		})
		return
	}

	portalInvoices := make([]portalInvoice, 0, len(invoices))
	// Class and admission details live on the invoice detail, so reuse them for the student list.
	studentDetails := map[string]portalStudent{}
	for i := range invoices {
		detail, err := models.GetInvoiceDetail(h.dbx, claims.SchoolID, invoices[i].ID)
		if err != nil {
			h.logger.WithError(err).
				WithField("invoice_id", invoices[i].ID).
				Warn("Skipping invoice that failed to load for portal")
			continue
		}
		portalInvoices = append(portalInvoices, toPortalInvoice(detail, school.Name))
		studentDetails[detail.StudentName] = portalStudent{
			AdmissionNo: detail.AdmissionNo,
			ClassName:   detail.ClassName,
		}
	}

	students := make([]portalStudent, 0, len(guardianDetail.Students))
	for _, link := range guardianDetail.Students {
		if link.Student == nil {
			continue
		}
		student := portalStudent{
			ID:        link.Student.ID,
			FirstName: link.Student.FirstName,
			LastName:  link.Student.LastName,
		}
		if extra, found := studentDetails[student.FirstName+" "+student.LastName]; found {
			student.AdmissionNo = extra.AdmissionNo
			student.ClassName = extra.ClassName
		}
		students = append(students, student)
	}

	expiresAt := ""
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time.Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, portalSessionResponse{
		Token:     c.Param("token"),
		ExpiresAt: expiresAt,
		Guardian: portalGuardian{
			ID:        guardianDetail.ID,
			FirstName: guardianDetail.FirstName,
			LastName:  guardianDetail.LastName,
			Email:     guardianDetail.Email,
			Phone:     guardianDetail.Phone,
		},
		School: portalSchool{
			ID:                school.ID,
			Name:              school.Name,
			Phone:             school.Phone,
			Email:             school.Email,
			Address:           school.Address,
			LogoURL:           derefString(school.LogoURL),
			BankName:          derefString(school.BankName),
			BankAccountName:   derefString(school.BankAccountName),
			BankAccountNumber: derefString(school.BankAccountNumber),
		},
		Students: students,
		Invoices: portalInvoices,
	})
}

// GetPortalInvoice returns a single invoice belonging to the link's guardian.
func (h *Handler) GetPortalInvoice(c *gin.Context) {
	claims, ok := h.authorizePortalRequest(c)
	if !ok {
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_invoice_id",
			Message: "Invalid invoice reference.",
		})
		return
	}

	detail, err := h.loadPortalInvoice(claims, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "invoice_not_found",
				Message: "This invoice is not available on your link.",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to load portal invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load this invoice",
		})
		return
	}

	school := models.School{BaseModel: models.BaseModel{ID: claims.SchoolID}}
	_ = school.GetProfile(h.dbx)

	c.JSON(http.StatusOK, toPortalInvoice(detail, school.Name))
}

// GetPortalInvoicePDF streams the invoice PDF for a guardian link.
func (h *Handler) GetPortalInvoicePDF(c *gin.Context) {
	claims, ok := h.authorizePortalRequest(c)
	if !ok {
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_invoice_id",
			Message: "Invalid invoice reference.",
		})
		return
	}

	if _, err := h.loadPortalInvoice(claims, invoiceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "invoice_not_found",
				Message: "This invoice is not available on your link.",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to authorize portal invoice PDF")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load this invoice",
		})
		return
	}

	pdfCtx, err := models.GetInvoicePDFContext(h.dbx, claims.SchoolID, invoiceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to load invoice for portal PDF")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load this invoice",
		})
		return
	}

	pdfBytes, err := pdflib.GenerateInvoicePDF(pdflib.FromInvoiceContext(pdfCtx))
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate portal invoice PDF")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "pdf_error",
			Message: "Failed to generate the invoice PDF",
		})
		return
	}

	filename := pdflib.Filename(pdfCtx.Invoice.InvoiceNo)
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// InitializePortalPayment starts an online payment for an invoice on a guardian link.
func (h *Handler) InitializePortalPayment(c *gin.Context) {
	claims, ok := h.authorizePortalRequest(c)
	if !ok {
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_invoice_id",
			Message: "Invalid invoice reference.",
		})
		return
	}

	var req PortalPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Enter a valid payment amount.",
		})
		return
	}

	detail, err := h.loadPortalInvoice(claims, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "invoice_not_found",
				Message: "This invoice is not available on your link.",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to load invoice for portal payment")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load this invoice",
		})
		return
	}

	if detail.AmountDue <= 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invoice_already_paid",
			Message: "This invoice has no outstanding balance.",
		})
		return
	}

	if h.config.PaystackSecretKey == "" {
		h.logger.
			WithField("invoice_id", invoiceID).
			Warn("Portal payment attempted without Paystack configuration")
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "payment_unavailable",
			Message: "Online payment is not enabled yet. Please pay using the school's bank details shown on your invoice.",
		})
		return
	}

	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "payment_not_implemented",
		Message: "Online payment is being set up. Please pay using the school's bank details shown on your invoice.",
	})
}

// CreatePortalGraceRequest emails the school a guardian's grace-period meeting request.
func (h *Handler) CreatePortalGraceRequest(c *gin.Context) {
	claims, ok := h.authorizePortalRequest(c)
	if !ok {
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_invoice_id",
			Message: "Invalid invoice reference.",
		})
		return
	}

	var req PortalGraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Choose a preferred date and time, and tell the school why you need more time.",
		})
		return
	}

	detail, err := h.loadPortalInvoice(claims, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "invoice_not_found",
				Message: "This invoice is not available on your link.",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to load invoice for grace request")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load this invoice",
		})
		return
	}

	school := models.School{BaseModel: models.BaseModel{ID: claims.SchoolID}}
	if err := school.GetProfile(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to load school for grace request")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load school details",
		})
		return
	}

	guardian := models.Guardian{ID: claims.GuardianID}
	if err := guardian.Get(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to load guardian for grace request")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load your details",
		})
		return
	}

	contactPhone := strings.TrimSpace(req.ContactPhone)
	if contactPhone == "" {
		contactPhone = guardian.Phone
	}

	graceReq := mail.GraceMeetingRequest{
		SchoolName:    school.Name,
		GuardianName:  guardian.FirstName + " " + guardian.LastName,
		GuardianEmail: guardian.Email,
		GuardianPhone: contactPhone,
		StudentName:   detail.StudentName,
		InvoiceNo:     detail.InvoiceNo,
		AmountDue:     detail.AmountDue,
		PreferredDate: req.PreferredDate,
		PreferredTime: req.PreferredTime,
		Reason:        req.Reason,
	}

	if err := mail.SendGraceMeetingRequestEmail(school.Email, graceReq); err != nil {
		h.logger.WithError(err).
			WithField("invoice_id", invoiceID).
			Error("Failed to email guardian grace request")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "grace_request_failed",
			Message: "We could not send your request. Please call the school directly.",
		})
		return
	}

	h.logger.
		WithField("school_id", claims.SchoolID).
		WithField("guardian_id", claims.GuardianID).
		WithField("invoice_id", invoiceID).
		Info("Guardian requested a grace period meeting")

	c.JSON(http.StatusOK, gin.H{
		"message": "Your meeting request for " + req.PreferredDate + " at " + req.PreferredTime +
			" has been sent to " + school.Name + ". The school will contact you shortly.",
	})
}
