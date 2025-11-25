package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ResendProvider struct {
	APIKey string
}

func (p *ResendProvider) SendEmailNotification(to, subject, message string) error {
	url := "https://api.resend.com/emails"

	payload := map[string]interface{}{
		"from":    "Notification Service <no-reply@resend.dev>",
		"to":      []string{to},
		"subject": subject,
		"html": fmt.Sprintf(`
			<h3>%s</h3>
			<p>%s</p>
		`, subject, message),
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
		respBytes, _ := io.ReadAll(resp.Body)
		log.Println("❌ Resend response:", string(respBytes))
		return fmt.Errorf("resend API returned %d", resp.StatusCode)
	}

	return nil
}
