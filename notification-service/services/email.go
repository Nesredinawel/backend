package services

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"notification-service/config"
	"notification-service/models"
)

func SendEmailNotification(n models.Notification, to string) {
	cfg := config.LoadConfig()
	if !cfg.EnableEmail || cfg.SendgridAPIKey == "" {
		return
	}

	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{{"email": to}},
			},
		},
		"from":    map[string]string{"email": "noreply@yourdomain.com"},
		"subject": n.Title,
		"content": []map[string]string{
			{"type": "text/plain", "value": n.Message},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.SendgridAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ SendGrid request error: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		log.Printf("⚠️ SendGrid failed: %s | body: %s", resp.Status, string(respBody))
	} else {
		log.Printf("✅ SendGrid sent email to %s | status: %s", to, resp.Status)
	}
}
