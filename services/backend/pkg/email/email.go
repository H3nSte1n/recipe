package email

import (
	"fmt"
	"net/smtp"
)

type EmailService interface {
	SendPasswordResetEmail(to, resetToken string) error
	SendVerificationEmail(to, verificationToken string) error
}

type emailService struct {
	from        string
	password    string
	host        string
	port        string
	frontendUrl string
}

func NewEmailService(from, password, host, port, frontendUrl string) EmailService {
	return &emailService{
		from:        from,
		password:    password,
		host:        host,
		port:        port,
		frontendUrl: frontendUrl,
	}
}

func (s *emailService) SendPasswordResetEmail(to, resetToken string) error {
	subject := "Password Reset Request"
	resetLink := fmt.Sprintf("http://%s/reset-password?token=%s", s.frontendUrl, resetToken)
	body := fmt.Sprintf(`
        Hello,
        
        You have requested to reset your password. Click the link below to reset it:
        
        %s
        
        If you didn't request this, please ignore this email.
        
        The link will expire in 1 hour.
        
        Best regards,
        Your App Team
    `, resetLink)

	msg := fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", to, subject, body)

	auth := smtp.PlainAuth("", s.from, s.password, s.host)
	return smtp.SendMail(
		s.host+":"+s.port,
		auth,
		s.from,
		[]string{to},
		[]byte(msg),
	)
}

func (s *emailService) SendVerificationEmail(to, verificationToken string) error {
	subject := "Verify Your Email Address"
	verifyLink := fmt.Sprintf("http://%s/verify-email?token=%s", s.frontendUrl, verificationToken)
	body := fmt.Sprintf(`
        Hello,

        Thanks for signing up. Please verify your email address by clicking the link below:

        %s

        If you didn't create this account, please ignore this email.

        The link will expire in 24 hours.

        Best regards,
        Your App Team
    `, verifyLink)

	msg := fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", to, subject, body)

	auth := smtp.PlainAuth("", s.from, s.password, s.host)
	return smtp.SendMail(
		s.host+":"+s.port,
		auth,
		s.from,
		[]string{to},
		[]byte(msg),
	)
}
