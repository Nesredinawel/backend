package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type BrevoProvider struct {
	APIKey string
	From   string // Example: "yourname@gmail.com"
}

func (p *BrevoProvider) SendVerificationEmail(to, verificationURL string) error {
	url := "https://api.brevo.com/v3/smtp/email"

	payload := map[string]interface{}{
		"sender": map[string]string{
			"email": p.From,
			"name":  "Auth Service",
		},
		"to": []map[string]string{
			{"email": to},
		},
		"subject": "Confirm your email address",
		"htmlContent": fmt.Sprintf(`
			<h2>Confirm your email</h2>
			<p>Click below to verify your email:</p>
			<a href="%s" target="_blank">Verify Email</a>
			<p>This link expires in 15 minutes.</p>
		`, verificationURL),
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("api-key", p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Println("❌ Brevo error:", resp.Status)
		return fmt.Errorf("brevo returned %d", resp.StatusCode)
	}

	return nil
}
