package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/jung-kurt/gofpdf/v2"
	"github.com/shopspring/decimal"
)

const (
	defaultProviderName = "School Invoice"
	defaultProviderURL  = "https://schoolinvoice.app"
)

// LineItem is a single row on the invoice.
type LineItem struct {
	Description string
	Quantity    int
	UnitPrice   float64
	Amount      float64
}

// InvoiceData contains everything needed to render the PDF.
type InvoiceData struct {
	// School (issuer)
	SchoolName    string
	SchoolAddress string
	SchoolPhone   string
	SchoolEmail   string

	// Bank / payment details (optional)
	BankName          string
	BankAccountName   string
	BankAccountNumber string

	// Recipient
	RecipientName    string
	RecipientAddress string

	// Student
	StudentName string
	AdmissionNo string
	ClassName   string

	// Invoice meta
	InvoiceNo     string
	IssueDate     time.Time
	DueDate       time.Time
	ReferenceNo   string
	TotalAmount   float64
	AmountPaid    float64
	Currency      string
	LineItems     []LineItem

	// Service provider branding (shown in corner)
	ProviderName string
	ProviderURL  string
}

func providerName() string {
	if v := os.Getenv("SERVICE_PROVIDER_NAME"); v != "" {
		return v
	}
	return defaultProviderName
}

func providerURL() string {
	if v := os.Getenv("SERVICE_PROVIDER_URL"); v != "" {
		return v
	}
	return defaultProviderURL
}

// GenerateInvoicePDF renders an invoice PDF in the style of a European payment slip with barcode.
func GenerateInvoicePDF(data InvoiceData) ([]byte, error) {
	if data.Currency == "" {
		data.Currency = "NGN"
	}
	if data.ProviderName == "" {
		data.ProviderName = providerName()
	}
	if data.ProviderURL == "" {
		data.ProviderURL = providerURL()
	}
	if data.ReferenceNo == "" {
		data.ReferenceNo = FormatReferenceNo(data.InvoiceNo)
	}
	if data.IssueDate.IsZero() {
		data.IssueDate = time.Now()
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 12, 15)
	pdf.AddPage()
	pdf.SetAutoPageBreak(false, 15)

	// Service provider — vertical text in left margin (like myClub in reference)
	pdf.SetFont("Arial", "", 6)
	pdf.SetTextColor(140, 140, 140)
	pdf.TransformBegin()
	pdf.TransformRotate(90, 8, 150)
	pdf.Text(8, 150, fmt.Sprintf("%s — %s", data.ProviderName, data.ProviderURL))
	pdf.TransformEnd()
	pdf.SetTextColor(0, 0, 0)

	// Header — school info (left) + INVOICE title (right)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(100, 6, data.SchoolName, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	if data.SchoolAddress != "" {
		pdf.CellFormat(100, 5, data.SchoolAddress, "", 1, "L", false, 0, "")
	}
	contact := strings.TrimSpace(strings.Join([]string{data.SchoolPhone, data.SchoolEmail}, " · "))
	if contact != "" {
		pdf.CellFormat(100, 5, contact, "", 1, "L", false, 0, "")
	}

	pdf.SetXY(130, 12)
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(65, 10, "INVOICE", "", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.SetX(130)
	pdf.CellFormat(65, 4, fmt.Sprintf("Page 1/1"), "", 1, "R", false, 0, "")

	y := pdf.GetY() + 4

	// Recipient block
	pdf.SetXY(15, y)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(100, 5, data.RecipientName, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	if data.RecipientAddress != "" {
		for _, line := range strings.Split(data.RecipientAddress, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				pdf.CellFormat(100, 4.5, line, "", 1, "L", false, 0, "")
			}
		}
	}

	// Metadata table (right side)
	metaY := y
	meta := []struct{ label, value string }{
		{"Student", data.StudentName},
		{"Admission No", data.AdmissionNo},
		{"Class", data.ClassName},
		{"Invoice No", data.InvoiceNo},
		{"Issue Date", formatDate(data.IssueDate)},
		{"Due Date", formatDate(data.DueDate)},
		{"Reference No", formatRefDisplay(data.ReferenceNo)},
		{"Delivery", "Email"},
	}
	pdf.SetFont("Arial", "", 8)
	for i, row := range meta {
		if row.value == "" {
			continue
		}
		pdf.SetXY(115, metaY+float64(i)*5)
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(30, 5, row.label, "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(50, 5, row.value, "", 0, "R", false, 0, "")
	}

	pdf.SetY(y + 28)

	// Line items table
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(245, 245, 245)
	headers := []struct{ w float64; label string }{
		{75, "Description"},
		{15, "Qty"},
		{25, "Unit Price"},
		{25, "VAT %"},
		{25, "Excl. VAT"},
		{25, "Incl. VAT"},
	}
	for _, h := range headers {
		pdf.CellFormat(h.w, 7, h.label, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Arial", "", 8)

	for _, item := range data.LineItems {
		qty := item.Quantity
		if qty < 1 {
			qty = 1
		}
		unit := item.UnitPrice
		if unit == 0 {
			unit = item.Amount
		}
		pdf.CellFormat(75, 6, truncate(item.Description, 42), "LR", 0, "L", false, 0, "")
		pdf.CellFormat(15, 6, fmt.Sprintf("%d", qty), "R", 0, "C", false, 0, "")
		pdf.CellFormat(25, 6, formatMoney(unit), "R", 0, "R", false, 0, "")
		pdf.CellFormat(25, 6, "0.0", "R", 0, "R", false, 0, "")
		pdf.CellFormat(25, 6, formatMoney(item.Amount), "R", 0, "R", false, 0, "")
		pdf.CellFormat(25, 6, formatMoney(item.Amount), "R", 1, "R", false, 0, "")
	}

	amountDue := data.TotalAmount - data.AmountPaid
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(165, 7, "Total due", "T", 0, "R", false, 0, "")
	pdf.CellFormat(25, 7, formatMoney(amountDue), "T", 1, "R", false, 0, "")

	// Notice
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 8)
	notice := fmt.Sprintf(
		"Please pay the amount shown below by %s. Reference number must be quoted with payment. "+
			"Contact %s for questions about this invoice.",
		formatDate(data.DueDate), data.SchoolEmail,
	)
	pdf.MultiCell(180, 4, notice, "", "L", false)

	// Payment slip section (dashed separator)
	slipY := 220.0
	pdf.SetY(slipY)
	pdf.SetDrawColor(180, 180, 180)
	for x := 15.0; x < 195; x += 3 {
		pdf.Line(x, slipY, x+1.5, slipY)
	}
	pdf.SetDrawColor(0, 0, 0)

	slipY += 4
	pdf.SetXY(15, slipY)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(90, 4, data.SchoolName, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(90, 4, data.SchoolAddress, "", 1, "L", false, 0, "")
	if data.SchoolEmail != "" {
		pdf.CellFormat(90, 4, data.SchoolEmail, "", 1, "L", false, 0, "")
	}

	// Payee / payer columns
	colY := slipY
	pdf.SetXY(15, colY+14)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(20, 4, "Payee", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 7)
	payee := data.SchoolName
	if data.BankAccountName != "" {
		payee = data.BankAccountName
	}
	pdf.CellFormat(70, 4, payee, "", 1, "L", false, 0, "")

	if data.BankAccountNumber != "" {
		pdf.SetX(15)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(20, 4, "Account", "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 7)
		acct := data.BankAccountNumber
		if data.BankName != "" {
			acct = data.BankName + " — " + acct
		}
		pdf.CellFormat(70, 4, acct, "", 1, "L", false, 0, "")
	}

	pdf.SetX(15)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(20, 4, "Payer", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(70, 4, data.RecipientName, "", 1, "L", false, 0, "")

	// Right column — reference, due date, amount
	pdf.SetXY(110, colY+14)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(35, 4, "Reference No", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(50, 4, formatRefDisplay(data.ReferenceNo), "", 1, "R", false, 0, "")

	pdf.SetX(110)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(35, 4, "Due Date", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(50, 4, formatDate(data.DueDate), "", 1, "R", false, 0, "")

	pdf.SetX(110)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(35, 4, "Amount", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(50, 4, fmt.Sprintf("%s %s", data.Currency, formatMoney(amountDue)), "", 1, "R", false, 0, "")

	// Barcode
	barcodePayload := BuildBarcodePayload(data.ReferenceNo, amountDue, data.DueDate)
	if err := drawBarcode(pdf, barcodePayload, 15, 268, 180, 14); err != nil {
		return nil, err
	}
	pdf.SetXY(15, 283)
	pdf.SetFont("Arial", "", 6)
	pdf.CellFormat(180, 4, barcodePayload, "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FormatReferenceNo builds a numeric reference from an invoice number.
func FormatReferenceNo(invoiceNo string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, invoiceNo)
	if digits == "" {
		digits = "00000000000"
	}
	for len(digits) < 15 {
		digits = "0" + digits
	}
	if len(digits) > 20 {
		digits = digits[:20]
	}
	return digits
}

// BuildBarcodePayload creates a scannable payment reference string.
func BuildBarcodePayload(reference string, amount float64, dueDate time.Time) string {
	ref := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, reference)
	cents := int64(decimal.NewFromFloat(amount).Mul(decimal.NewFromInt(100)).IntPart())
	due := dueDate.Format("060102")
	return fmt.Sprintf("4%015s%010d%s", ref, cents, due)
}

func formatRefDisplay(ref string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, ref)
	var parts []string
	for i := 0; i < len(digits); i += 5 {
		end := i + 5
		if end > len(digits) {
			end = len(digits)
		}
		parts = append(parts, digits[i:end])
	}
	return strings.Join(parts, " ")
}

func formatDate(t time.Time) string {
	return t.Format("2.1.2006")
}

func formatMoney(v float64) string {
	return strings.Replace(fmt.Sprintf("%.2f", v), ".", ",", -1)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func drawBarcode(pdf *gofpdf.Fpdf, payload string, x, y, width, height float64) error {
	bc, err := code128.Encode(payload)
	if err != nil {
		return err
	}
	scaled, err := barcode.Scale(bc, int(width*8), int(height*8))
	if err != nil {
		return err
	}
	// gofpdf only supports 8-bit PNG; barcode images may be 16-bit grayscale.
	rgba := imageToRGBA(scaled)
	var imgBuf bytes.Buffer
	if err := png.Encode(&imgBuf, rgba); err != nil {
		return err
	}
	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	name := "invoice_barcode"
	pdf.RegisterImageOptionsReader(name, opt, &imgBuf)
	pdf.ImageOptions(name, x, y, width, height, false, opt, 0, "")
	return nil
}

func imageToRGBA(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
	return dst
}
