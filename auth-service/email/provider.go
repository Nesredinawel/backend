package email

type EmailProvider interface {
	SendVerificationEmail(to, verificationURL string) error
}
