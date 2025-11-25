package handlers

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

// SwaggerUI serves Swagger UI (optional, can use Redoc instead)
func SwaggerUI(specURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers for browser Swagger UI
		addCORSHeaders(w, r)

		handler := httpSwagger.Handler(
			httpSwagger.URL(specURL),
		)
		handler.ServeHTTP(w, r)
	}
}

// OpenAPISpec serves YAML files with no caching
func OpenAPISpec(filepath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ✅ CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Surrogate-Control", "no-store")

		http.ServeFile(w, r, filepath)
	}
}

// RedocUI serves a simple HTML page loading Redoc
func RedocUI(specURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers
		addCORSHeaders(w, r)

		html := `<!DOCTYPE html>
<html>
<head>
  <title>API Docs</title>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</head>
<body>
  <redoc spec-url="` + specURL + `"></redoc>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}
}

// addCORSHeaders allows cross-origin requests for Swagger/Redoc
func addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Or restrict to Swagger UI origin
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	// Handle preflight requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
}
