package main

import (
	"os"
	"sync"
	"testing"

	"sentinel/internal/rediskv"
	"sentinel/internal/testredis"
)

func TestLoginAllowed_LimitsByIPAndEmail(t *testing.T) {
	oldUnsafe := os.Getenv("DEV_UNSAFE")
	t.Cleanup(func() {
		_ = os.Setenv("DEV_UNSAFE", oldUnsafe)
		loginAttempts = sync.Map{}
	})
	_ = os.Setenv("DEV_UNSAFE", "true")
	loginAttempts = sync.Map{}
	ip := "10.0.0.1"
	email := "user@example.com"
	for i := 0; i < 5; i++ {
		blocked, err := loginAttemptsExceeded(ip, email)
		if err != nil {
			t.Fatalf("unexpected error on attempt %d: %v", i+1, err)
		}
		if blocked {
			t.Fatalf("attempt %d unexpectedly blocked", i+1)
		}
		if err := recordFailedLogin(ip, email); err != nil {
			t.Fatalf("record failed login: %v", err)
		}
	}
	blocked, err := loginAttemptsExceeded(ip, email)
	if err != nil {
		t.Fatalf("unexpected error after threshold: %v", err)
	}
	if !blocked {
		t.Fatalf("expected limiter to block after 5 failed attempts")
	}
}

func TestLoginAllowed_SeparatesDifferentEmails(t *testing.T) {
	oldUnsafe := os.Getenv("DEV_UNSAFE")
	t.Cleanup(func() {
		_ = os.Setenv("DEV_UNSAFE", oldUnsafe)
		loginAttempts = sync.Map{}
	})
	_ = os.Setenv("DEV_UNSAFE", "true")
	loginAttempts = sync.Map{}
	ip := "10.0.0.1"
	for i := 0; i < 5; i++ {
		if err := recordFailedLogin(ip, "first@example.com"); err != nil {
			t.Fatalf("record failed login: %v", err)
		}
	}
	blocked, err := loginAttemptsExceeded(ip, "second@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocked {
		t.Fatalf("different email should have independent quota")
	}
}

func TestLoginAllowed_UsesRedisWhenEnabled(t *testing.T) {
	srv, err := testredis.Start()
	if err != nil {
		t.Fatalf("start redis stub: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	oldAddr := os.Getenv("REDIS_ADDR")
	oldEnabled := os.Getenv("REDIS_ENABLED")
	t.Cleanup(func() {
		_ = os.Setenv("REDIS_ADDR", oldAddr)
		_ = os.Setenv("REDIS_ENABLED", oldEnabled)
		loginRedis = rediskv.NewFromEnv()
		loginAttempts = sync.Map{}
	})

	_ = os.Setenv("REDIS_ADDR", srv.Addr())
	_ = os.Setenv("REDIS_ENABLED", "true")
	loginRedis = rediskv.NewFromEnv()
	loginAttempts = sync.Map{}

	for i := 0; i < 5; i++ {
		blocked, err := loginAttemptsExceeded("127.0.0.1", "redis@example.com")
		if err != nil {
			t.Fatalf("attempt %d returned error: %v", i+1, err)
		}
		if blocked {
			t.Fatalf("attempt %d unexpectedly blocked", i+1)
		}
		if err := recordFailedLogin("127.0.0.1", "redis@example.com"); err != nil {
			t.Fatalf("record failed login: %v", err)
		}
	}
	blocked, err := loginAttemptsExceeded("127.0.0.1", "redis@example.com")
	if err != nil {
		t.Fatalf("unexpected error after threshold: %v", err)
	}
	if !blocked {
		t.Fatalf("expected redis-backed limiter to block after 5 failed attempts")
	}
}

func TestLoginAllowed_FailsClosedWithoutRedisWhenUnsafeDisabled(t *testing.T) {
	oldAddr := os.Getenv("REDIS_ADDR")
	oldEnabled := os.Getenv("REDIS_ENABLED")
	oldUnsafe := os.Getenv("DEV_UNSAFE")
	t.Cleanup(func() {
		_ = os.Setenv("REDIS_ADDR", oldAddr)
		_ = os.Setenv("REDIS_ENABLED", oldEnabled)
		_ = os.Setenv("DEV_UNSAFE", oldUnsafe)
		loginRedis = rediskv.NewFromEnv()
		loginAttempts = sync.Map{}
	})

	_ = os.Setenv("REDIS_ADDR", "127.0.0.1:1")
	_ = os.Setenv("REDIS_ENABLED", "true")
	_ = os.Setenv("DEV_UNSAFE", "false")
	loginRedis = rediskv.NewFromEnv()
	loginAttempts = sync.Map{}

	if _, err := loginAttemptsExceeded("127.0.0.1", "blocked@example.com"); err == nil {
		t.Fatalf("expected missing redis to fail closed")
	}
	if err := recordFailedLogin("127.0.0.1", "blocked@example.com"); err == nil {
		t.Fatalf("expected recordFailedLogin to fail closed without redis")
	}
}
