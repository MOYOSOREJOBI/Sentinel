package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"sentinel/internal/rediskv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"sentinel/internal/audit"
	"sentinel/internal/auth"
	"sentinel/internal/config"
	"sentinel/internal/db"
	"sentinel/internal/healthcheck"
	"sentinel/internal/httpx"
	"sentinel/internal/metrics"
	"sentinel/internal/middleware"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		port := healthcheck.MustPort("PORT", 8080)
		os.Exit(healthcheck.Run(port, "/readyz"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := config.Load("gateway-api")
	pool, err := db.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	priv, _ := auth.ReadPrivate(cfg.JWTPrivateKey)
	pub, _ := auth.ReadPublic(cfg.JWTPublicKey)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS)
	r.Use(metrics.HTTPMiddleware("gateway_api"))
	r.Use(middleware.RequireCSRFFunc(func(r *http.Request) (string, bool) {
		claims, ok := authn(r, pub)
		if !ok {
			return "", false
		}
		return claims.Subject, true
	}))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"service": "gateway-api",
			"version": "dev",
			"links": map[string]string{
				"healthz": "/healthz",
				"readyz":  "/readyz",
				"metrics": "/metrics",
				"docs":    "/docs",
			},
		})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if pool.Ping(r.Context()) != nil {
			http.Error(w, "not ready", 503)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/metrics", metrics.Handler)

	r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Email, Password string }
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		if loginAttemptsExceeded(clientIP(r), in.Email) {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			return
		}
		var hash, role string
		err := pool.QueryRow(r.Context(), `SELECT password_hash,role FROM users WHERE email=$1`, in.Email).Scan(&hash, &role)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
			recordFailedLogin(clientIP(r), in.Email)
			http.Error(w, "unauthorized", 401)
			return
		}
		clearLoginAttempts(clientIP(r), in.Email)
		tok, err := auth.Sign(in.Email, role, priv)
		if err != nil {
			http.Error(w, "internal", 500)
			return
		}
		csrf, err := middleware.NewCSRFToken()
		if err != nil {
			http.Error(w, "internal", 500)
			return
		}
		middleware.BindCSRF(in.Email, csrf, 30*time.Minute)
		secure := cookieSecure(r)
		http.SetCookie(w, &http.Cookie{Name: "sentinel_token", Value: tok, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Path: "/", Expires: time.Now().Add(24 * time.Hour)})
		http.SetCookie(w, &http.Cookie{Name: "sentinel_csrf", Value: csrf, HttpOnly: false, Secure: secure, SameSite: http.SameSiteLaxMode, Path: "/", Expires: time.Now().Add(30 * time.Minute)})
		_ = audit.Append(r.Context(), pool, in.Email, "login", "success")
		httpx.JSON(w, 200, map[string]any{"role": role})
	})

	r.Post("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := authn(r, pub)
		if ok {
			middleware.ClearCSRF(claims.Subject)
		}
		secure := cookieSecure(r)
		http.SetCookie(w, &http.Cookie{Name: "sentinel_token", Value: "", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Path: "/", MaxAge: -1})
		http.SetCookie(w, &http.Cookie{Name: "sentinel_csrf", Value: "", HttpOnly: false, Secure: secure, SameSite: http.SameSiteLaxMode, Path: "/", MaxAge: -1})
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := authn(r, pub)
		if !ok {
			http.Error(w, "unauthorized", 401)
			return
		}
		httpx.JSON(w, 200, map[string]any{"user": claims.Subject, "role": claims.Role})
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	_ = srv.Shutdown(context.Background())
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	return r.RemoteAddr
}

func authn(r *http.Request, pub *rsa.PublicKey) (*auth.Claims, bool) {
	c, err := r.Cookie("sentinel_token")
	if err != nil {
		return nil, false
	}
	claims, err := auth.Parse(c.Value, pub)
	if err != nil {
		return nil, false
	}
	return claims, true
}

type loginLimitEntry struct {
	Count int
	Reset time.Time
}

var loginAttempts sync.Map
var loginRedis = rediskv.NewFromEnv()

func loginKey(ip, email string) string {
	return "login:" + strings.ToLower(strings.TrimSpace(email)) + ":" + ip
}

func loginAttemptsExceeded(ip, email string) bool {
	key := loginKey(ip, email)
	if loginRedis != nil && loginRedis.Enabled() {
		raw, ok, err := loginRedis.Get(context.Background(), key)
		if err == nil && ok {
			n, convErr := strconv.Atoi(strings.TrimSpace(raw))
			if convErr == nil {
				return n >= 5
			}
		}
	}
	now := time.Now()
	v, ok := loginAttempts.Load(key)
	if !ok {
		return false
	}
	e := v.(loginLimitEntry)
	if now.After(e.Reset) {
		loginAttempts.Delete(key)
		return false
	}
	return e.Count >= 5
}

func recordFailedLogin(ip, email string) {
	key := "login:" + strings.ToLower(strings.TrimSpace(email)) + ":" + ip
	window := 5 * time.Minute
	if loginRedis != nil && loginRedis.Enabled() {
		n, err := loginRedis.Incr(context.Background(), key)
		if err == nil {
			if n == 1 {
				_ = loginRedis.Expire(context.Background(), key, window)
			}
			return
		}
	}
	now := time.Now()
	v, _ := loginAttempts.LoadOrStore(key, loginLimitEntry{Count: 0, Reset: now.Add(window)})
	e := v.(loginLimitEntry)
	if now.After(e.Reset) {
		e = loginLimitEntry{Count: 0, Reset: now.Add(window)}
	}
	e.Count++
	loginAttempts.Store(key, e)
}

func clearLoginAttempts(ip, email string) {
	key := loginKey(ip, email)
	loginAttempts.Delete(key)
	if loginRedis != nil && loginRedis.Enabled() {
		_ = loginRedis.Del(context.Background(), key)
	}
}

func cookieSecure(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "local") {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "http") {
		return false
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	host = strings.ToLower(host)
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		return false
	}
	return true
}
