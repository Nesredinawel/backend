package email

type EmailProvider interface {
	SendVerificationEmail(to, verificationURL string) error
}

type NoopProvider struct{}

func (p *NoopProvider) SendVerificationEmail(to, verificationURL string) error {
	return nil
}
