package pdf

import (
	"time"

	"github.com/school-invoice/backend/internal/models"
)

// FromInvoiceContext maps a database invoice context to PDF render data.
func FromInvoiceContext(ctx *models.InvoicePDFContext) InvoiceData {
	total, _ := ctx.Invoice.TotalAmount.Float64()
	paid, _ := ctx.Invoice.AmountPaid.Float64()

	items := make([]LineItem, 0, len(ctx.Items))
	for _, item := range ctx.Items {
		amount, _ := item.Amount.Float64()
		items = append(items, LineItem{
			Description: item.Description,
			Quantity:    1,
			UnitPrice:   amount,
			Amount:      amount,
		})
	}

	bankName, bankAcctName, bankAcctNo := "", "", ""
	if ctx.School.BankName != nil {
		bankName = *ctx.School.BankName
	}
	if ctx.School.BankAccountName != nil {
		bankAcctName = *ctx.School.BankAccountName
	}
	if ctx.School.BankAccountNumber != nil {
		bankAcctNo = *ctx.School.BankAccountNumber
	}

	recipient := ctx.GuardianName
	if recipient == "" {
		recipient = ctx.StudentName
	}
	addr := ctx.GuardianAddr
	if addr == "" {
		addr = ctx.School.Address
	}

	return InvoiceData{
		SchoolName:        ctx.School.Name,
		SchoolAddress:     ctx.School.Address,
		SchoolPhone:       ctx.School.Phone,
		SchoolEmail:       ctx.School.Email,
		BankName:          bankName,
		BankAccountName:   bankAcctName,
		BankAccountNumber: bankAcctNo,
		RecipientName:     recipient,
		RecipientAddress:  addr,
		StudentName:       ctx.StudentName,
		AdmissionNo:       ctx.AdmissionNo,
		ClassName:         ctx.ClassName,
		InvoiceNo:         ctx.Invoice.InvoiceNo,
		IssueDate:         ctx.Invoice.CreatedAt,
		DueDate:           ctx.Invoice.DueDate,
		ReferenceNo:       FormatReferenceNo(ctx.Invoice.InvoiceNo),
		TotalAmount:       total,
		AmountPaid:        paid,
		Currency:          "NGN",
		LineItems:         items,
		ProviderName:      providerName(),
		ProviderURL:       providerURL(),
	}
}

// Filename returns a safe attachment filename for an invoice.
func Filename(invoiceNo string) string {
	safe := ""
	for _, r := range invoiceNo {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			safe += string(r)
		} else if r == ' ' {
			safe += "_"
		}
	}
	if safe == "" {
		safe = "invoice"
	}
	return safe + ".pdf"
}

// DefaultDueDays returns payment terms in days when not otherwise specified.
func DefaultDueDays(issue, due time.Time) int {
	days := int(due.Sub(issue).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
