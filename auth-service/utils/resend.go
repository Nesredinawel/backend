package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// SendVerificationEmail sends an email using Resend API with debug logs
func SendVerificationEmail(apiKey, recipientEmail, verificationURL string) error {
	url := "https://api.resend.com/emails"

	payload := map[string]interface{}{
		"from":    "Auth Service <no-reply@resend.dev>", // verified/test email
		"to":      []string{recipientEmail},
		"subject": "Confirm your email address",
		"html": fmt.Sprintf(`
        <h2>Confirm your email</h2>
        <p>Click below to verify your email and complete signup:</p>
        <a href="%s" target="_blank">Verify Email</a>
        <p>This link expires in 15 minutes.</p>
    `, verificationURL),
	}

	body, _ := json.Marshal(payload)
	log.Println("📩 Resend payload:", string(body)) // DEBUG: log request payload

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Println("❌ Failed to create request:", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	log.Println("🔑 Authorization header set with API key") // DEBUG

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("❌ Failed to send request to Resend:", err)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	log.Println("📬 Resend response status:", resp.Status)
	log.Println("📬 Resend response body:", string(respBody)) // DEBUG: log response body

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend API returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Println("✅ Verification email sent successfully to", recipientEmail)
	return nil
}
