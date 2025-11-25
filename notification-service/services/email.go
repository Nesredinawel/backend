package services

import (
	"log"
	"notification-service/email"
	"notification-service/models"
)

var provider = email.NewEmailProvider()

// SendEmailNotification sends a normal notification email (not verification)
func SendEmailNotification(n models.Notification, to string) {
	err := provider.SendEmailNotification(
		to,
		n.Title,
		n.Message,
	)

	if err != nil {
		log.Println("❌ Email send failed:", err)
	} else {
		log.Println("📧 Email sent successfully to", to)
	}
}
