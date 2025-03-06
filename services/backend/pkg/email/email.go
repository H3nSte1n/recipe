package email

import (
	"fmt"
	"net/smtp"
)

type EmailService interface {
	SendPasswordResetEmail(to, resetToken string) error
}

type emailService struct {
	from     string
	password string
	host     string
	port     string
}

func NewEmailService(from, password, host, port string) EmailService {
	return &emailService{
		from:     from,
		password: password,
		host:     host,
		port:     port,
	}
}

func (s *emailService) SendPasswordResetEmail(to, resetToken string) error {
	subject := "Password Reset Request"
	frontend_url := "localhost:3000"
	resetLink := fmt.Sprintf("http://%s/reset-password?token=%s", frontend_url, resetToken)
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
