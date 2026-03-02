package middleware

import (
	"net/http"
	"os"
	"strings"
)

func CORS(next http.Handler) http.Handler {
	allowed := loadAllowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			if origin == "" || !allowed[origin] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoadAllowedOrigins returns the map of allowed CORS origins from ALLOWED_ORIGINS env var.
// If not set and APP_ENV=local, defaults to http://localhost:3000,http://127.0.0.1:3000 .
func LoadAllowedOrigins() map[string]bool {
	out := map[string]bool{}
	v := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if v == "" && strings.ToLower(os.Getenv("APP_ENV")) == "local" {
		v = "http://localhost:3000,http://127.0.0.1:3000"
	}
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out[p] = true
		}
	}
	return out
}

func loadAllowedOrigins() map[string]bool {
	return LoadAllowedOrigins()
}
