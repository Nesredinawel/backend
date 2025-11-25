package email

import "os"

func NewEmailProvider() EmailProvider {
	switch os.Getenv("EMAIL_PROVIDER") {

	case "brevo":
		return &BrevoProvider{
			APIKey: os.Getenv("BREVO_API_KEY"),
			From:   os.Getenv("BREVO_FROM_EMAIL"),
		}

	case "resend":
		return &ResendProvider{
			APIKey: os.Getenv("RESEND_API_KEY"),
		}
	}

	panic("❌ invalid EMAIL_PROVIDER. Options: brevo, resend")
}
