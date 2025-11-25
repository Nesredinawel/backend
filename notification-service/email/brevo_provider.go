package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type BrevoProvider struct {
	APIKey string
	From   string
}

func (p *BrevoProvider) SendEmailNotification(to, subject, message string) error {
	url := "https://api.brevo.com/v3/smtp/email"

	payload := map[string]interface{}{
		"sender": map[string]string{
			"email": p.From,
			"name":  "Notification Service",
		},
		"to": []map[string]string{
			{"email": to},
		},
		"subject": subject,
		"htmlContent": fmt.Sprintf(`
			<h3>%s</h3>
			<p>%s</p>
		`, subject, message),
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
		respBytes, _ := io.ReadAll(resp.Body)
		log.Println("❌ Brevo response:", string(respBytes))
		return fmt.Errorf("brevo returned %d", resp.StatusCode)
	}

	return nil
}
