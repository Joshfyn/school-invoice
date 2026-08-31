package mail

import (
	"fmt"
	"io"
	"os"

	"github.com/school-invoice/backend/lib"
	pdflib "github.com/school-invoice/backend/lib/pdf"
	"gopkg.in/gomail.v2"
)

const (
	smtpHost = "smtp.gmail.com"
	smtpPort = 587

	// Base URL for the API endpoint that will handle the verification request
	scheme              = "http"
	confirmationBaseURL = "%s://%s/api/v1/auth/reset-password"
)

// SendConfirmationEmail sends a verification link containing the JWT.
func SendResetPasswordEmail(recipientEmail, jwtToken string) error {

	// 1. CONSTRUCT THE FULL VERIFICATION URL

	verificationURL := fmt.Sprintf("%s?token=%s", fmt.Sprintf(confirmationBaseURL, scheme, lib.GetServiceHostWithPort("api")), jwtToken)

	// 2. BUILD THE EMAIL CONTENT
	msg := gomail.NewMessage()
	msg.SetHeader("From", fromAddress())
	msg.SetHeader("To", recipientEmail)
	msg.SetHeader("Subject", "Action Required: Reset Your Password")

	// HTML body:
	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<p>School Invoice! Forgot your password? No problem. Please click the button below to reset your password.</p>
			
			<div style="margin-top: 20px;">
				<a href="%s" 
				   style="padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; font-weight: bold;">
					Reset My Password
				</a>
			</div>
			
			<p style="margin-top: 20px; font-size: 10px; color: #888;">
				If the button doesn't work, copy and paste this link into your browser:<br>
				<a href="%s">%s</a>
			</p>
			<p>The link will expire in 1 hour.</p>
		</body>
		</html>
	`, verificationURL, verificationURL, verificationURL)

	msg.SetBody("text/html", htmlBody)

	// get the smtp pass from the environment variable
	if err := dialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send confirmation email: %w", err)
	}
	return nil
}

// SendInvoiceEmail sends an invoice PDF and secure portal link to a guardian.
func SendInvoiceEmail(recipientEmail, schoolName, invoiceNo, studentName, portalURL string, pdfBytes []byte, isReminder bool) error {
	if recipientEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	subject := fmt.Sprintf("Invoice %s from %s", invoiceNo, schoolName)
	if isReminder {
		subject = fmt.Sprintf("Reminder: Invoice %s from %s", invoiceNo, schoolName)
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", fromAddress())
	msg.SetHeader("To", recipientEmail)
	msg.SetHeader("Subject", subject)

	body := fmt.Sprintf(`
		<html><body style="font-family: Arial, sans-serif; color: #333;">
			<p>Dear Parent/Guardian,</p>
			<p>Please find attached the invoice <strong>%s</strong> for <strong>%s</strong> from <strong>%s</strong>.</p>
			<p>Payment is due by the date shown on the invoice. Please quote the reference number when making payment.</p>
			<div style="margin: 24px 0;">
				<a href="%s"
				   style="display: inline-block; padding: 12px 20px; background-color: #059669; color: white; text-decoration: none; border-radius: 8px; font-weight: bold;">
					View invoice and pay online
				</a>
			</div>
			<p style="font-size: 12px; color: #666;">
				You can also use this secure link to view current and past invoices, download the invoice PDF, pay an outstanding balance, or request a grace-period meeting:<br>
				<a href="%s">%s</a>
			</p>
			<p style="font-size: 12px; color: #888;">For your security, do not forward this personal link. It expires after 7 days.</p>
			<p style="font-size: 12px; color: #888;">This email was sent via %s.</p>
		</body></html>
	`, invoiceNo, studentName, schoolName, portalURL, portalURL, portalURL, providerName())

	msg.SetBody("text/html", body)
	msg.Attach(pdflib.Filename(invoiceNo), gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(pdfBytes)
		return err
	}))

	return dialAndSend(msg)
}

// GraceMeetingRequest holds the details a guardian submits from the portal.
type GraceMeetingRequest struct {
	SchoolName    string
	GuardianName  string
	GuardianEmail string
	GuardianPhone string
	StudentName   string
	InvoiceNo     string
	AmountDue     float64
	PreferredDate string
	PreferredTime string
	Reason        string
}

// SendGraceMeetingRequestEmail notifies a school that a guardian wants to discuss a grace period.
func SendGraceMeetingRequestEmail(schoolEmail string, req GraceMeetingRequest) error {
	if schoolEmail == "" {
		return fmt.Errorf("school email is required")
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", fromAddress())
	msg.SetHeader("To", schoolEmail)
	msg.SetHeader("Subject", fmt.Sprintf("Grace period request for invoice %s", req.InvoiceNo))
	if req.GuardianEmail != "" {
		msg.SetHeader("Reply-To", req.GuardianEmail)
	}

	body := fmt.Sprintf(`
		<html><body style="font-family: Arial, sans-serif; color: #333;">
			<p>A parent/guardian has requested a meeting about a grace period.</p>
			<table cellpadding="6" style="border-collapse: collapse;">
				<tr><td><strong>Guardian</strong></td><td>%s</td></tr>
				<tr><td><strong>Email</strong></td><td>%s</td></tr>
				<tr><td><strong>Phone</strong></td><td>%s</td></tr>
				<tr><td><strong>Student</strong></td><td>%s</td></tr>
				<tr><td><strong>Invoice</strong></td><td>%s</td></tr>
				<tr><td><strong>Amount due</strong></td><td>NGN %.2f</td></tr>
				<tr><td><strong>Preferred date</strong></td><td>%s</td></tr>
				<tr><td><strong>Preferred time</strong></td><td>%s</td></tr>
			</table>
			<p><strong>Reason</strong><br>%s</p>
			<p style="font-size: 12px; color: #888;">Sent from the %s guardian portal via %s.</p>
		</body></html>
	`,
		req.GuardianName, req.GuardianEmail, req.GuardianPhone, req.StudentName,
		req.InvoiceNo, req.AmountDue, req.PreferredDate, req.PreferredTime,
		req.Reason, req.SchoolName, providerName(),
	)

	msg.SetBody("text/html", body)

	return dialAndSend(msg)
}

func fromAddress() string {
	if user := os.Getenv("SMTP_USER"); user != "" {
		return user
	}
	return "no-reply@schoolinvoice.app"
}

func providerName() string {
	if v := os.Getenv("SERVICE_PROVIDER_NAME"); v != "" {
		return v
	}
	return "School Invoice"
}

func dialAndSend(msg *gomail.Message) error {
	smtpPass := os.Getenv("SMTP_PASS")
	smtpUser := os.Getenv("SMTP_USER")
	if smtpPass == "" || smtpUser == "" {
		return fmt.Errorf("SMTP_PASS or SMTP_USER is not set")
	}
	dial := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)
	if err := dial.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}
