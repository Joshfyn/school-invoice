package mail

import (
	"fmt"
	"log"
	"os"

	"github.com/school-invoice/backend/lib"
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
	msg.SetHeader("From", "no-reply@yourdomain.com")
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
	smtpPass := os.Getenv("SMTP_PASS")
	smtpUser := os.Getenv("SMTP_USER")
	if smtpPass == "" || smtpUser == "" {
		return fmt.Errorf("SMTP_PASS or SMTP_USER is not set %s", smtpUser)
	}

	dial := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)
	if err := dial.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send confirmation email: %w", err)
	}

	log.Printf("Verification email successfully dispatched to %s", recipientEmail)
	return nil
}
