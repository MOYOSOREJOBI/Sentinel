package middleware

import (
	"log"
	"os"
	"strings"
	"sync"

	"sentinel/internal/rediskv"
)

var securityStoreLogs sync.Map

func allowUnsafeInMemorySecurity() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("DEV_UNSAFE")), "true") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("DEV_UNSAFE_ALLOW_INMEMORY_SECURITY")), "true")
}

func logSecurityStoreOnce(key, msg string) {
	if _, loaded := securityStoreLogs.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	log.Printf("security: %s", msg)
}

func ReloadSecurityBackendsFromEnv() {
	csrfRedis = rediskv.NewFromEnv()
	rlRedis = rediskv.NewFromEnv()
}
