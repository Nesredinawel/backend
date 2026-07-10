package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	httpSwagger "github.com/swaggo/http-swagger"
)

func getScheme(r *http.Request) string {
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		return "https"
	}
	return "http"
}

func getHost(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	return r.Host
}

// SwaggerUI serves Swagger UI (optional, can use Redoc instead)
func SwaggerUI(specURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w, r)

		resolvedURL := specURL
		if !strings.HasPrefix(specURL, "http") {
			baseURL := os.Getenv("PUBLIC_BASE_URL")
			if baseURL == "" {
				scheme := getScheme(r)
				host := getHost(r)
				baseURL = fmt.Sprintf("%s://%s", scheme, host)
			}
			resolvedURL = strings.TrimRight(baseURL, "/") + specURL
		}

		handler := httpSwagger.Handler(
			httpSwagger.URL(resolvedURL),
			httpSwagger.PersistAuthorization(true),
		)
		handler.ServeHTTP(w, r)
	}
}

// OpenAPISpec serves YAML files with no caching
func OpenAPISpec(filepath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		resolvedURL := specURL
		if !strings.HasPrefix(specURL, "http") {
			baseURL := os.Getenv("PUBLIC_BASE_URL")
			if baseURL == "" {
				scheme := getScheme(r)
				host := getHost(r)
				baseURL = fmt.Sprintf("%s://%s", scheme, host)
			}
			resolvedURL = strings.TrimRight(baseURL, "/") + specURL
		}

		addCORSHeaders(w, r)

		html := `<!DOCTYPE html>
<html>
<head>
  <title>API Docs</title>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</head>
<body>
  <redoc spec-url="` + resolvedURL + `"></redoc>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}
}

func addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
}
