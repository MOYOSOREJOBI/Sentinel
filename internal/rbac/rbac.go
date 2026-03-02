package rbac

import (
	"net/http"
	"strings"
)

var permissions = map[string]map[string]bool{
	"admin":   {"*": true, "governance:read": true},
	"analyst": {"alerts:write": true, "replay:write": true, "read": true},
	"viewer":  {"read": true},
}

func Allowed(role, perm string) bool {
	p, ok := permissions[strings.ToLower(role)]
	if !ok {
		return false
	}
	return p["*"] || p[perm]
}

func Require(role, perm string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !Allowed(role, perm) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
