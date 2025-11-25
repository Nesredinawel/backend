package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type ResendProvider struct {
	APIKey string
}

func (p *ResendProvider) SendVerificationEmail(to, verificationURL string) error {
	url := "https://api.resend.com/emails"

	payload := map[string]interface{}{
		"from":    "Auth Service <no-reply@resend.dev>",
		"to":      []string{to},
		"subject": "Confirm your email address",
		"html": fmt.Sprintf(`
            <h2>Confirm your email</h2>
            <p>Click below to verify your email:</p>
            <a href="%s" target="_blank">Verify Email</a>
            <p>This link expires in 15 minutes.</p>
        `, verificationURL),
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Println("❌ Resend request creation failed:", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("❌ Resend API error:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend API returned %d", resp.StatusCode)
	}

	return nil
}
