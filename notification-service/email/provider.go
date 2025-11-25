package email

type EmailProvider interface {
	SendEmailNotification(to, subject, message string) error
}
