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

// SendInvoiceEmail sends an invoice PDF to a guardian.
func SendInvoiceEmail(recipientEmail, schoolName, invoiceNo, studentName string, pdfBytes []byte, isReminder bool) error {
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
			<p style="font-size: 12px; color: #888;">This email was sent via %s.</p>
		</body></html>
	`, invoiceNo, studentName, schoolName, providerName())

	msg.SetBody("text/html", body)
	msg.Attach(pdflib.Filename(invoiceNo), gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(pdfBytes)
		return err
	}))

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
