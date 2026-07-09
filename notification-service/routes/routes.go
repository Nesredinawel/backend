package routes

import (
	"notification-service/config"
	"notification-service/handlers"
	middleware "notification-service/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r *chi.Mux) {
	cfg := config.LoadConfig()

	r.Group(func(protected chi.Router) {
		protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

		protected.Get("/notifications", handlers.GetNotificationsHandler)
		protected.Get("/notifications/count", handlers.UnreadCountHandler)
		protected.Post("/notifications/mark-read", handlers.MarkReadHandler)
		protected.Get("/ws", handlers.WebSocketHandler)
	})
}
