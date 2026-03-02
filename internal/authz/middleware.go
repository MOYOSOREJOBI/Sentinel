package authz

import (
	"context"
	"crypto/rsa"
	"errors"
	"net/http"
	"strings"

	"sentinel/internal/auth"
	"sentinel/internal/rbac"
)

type claimsContextKey struct{}

func Authenticate(r *http.Request, pub *rsa.PublicKey) (*auth.Claims, error) {
	if pub == nil {
		return nil, errors.New("missing public key")
	}
	if authz := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		token := strings.TrimSpace(authz[7:])
		if token != "" {
			return auth.Parse(token, pub)
		}
	}
	c, err := r.Cookie("sentinel_token")
	if err != nil {
		return nil, err
	}
	return auth.Parse(c.Value, pub)
}

func SubjectFromRequest(pub *rsa.PublicKey) func(*http.Request) (string, bool) {
	return func(r *http.Request) (string, bool) {
		claims, err := Authenticate(r, pub)
		if err != nil {
			return "", false
		}
		return claims.Subject, true
	}
}

func WithClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func Claims(r *http.Request) (*auth.Claims, bool) {
	v := r.Context().Value(claimsContextKey{})
	if v == nil {
		return nil, false
	}
	claims, ok := v.(*auth.Claims)
	return claims, ok && claims != nil
}

func RequireAuth(pub *rsa.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := Authenticate(r, pub)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

func RequireAnyPerm(pub *rsa.PublicKey, perms ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RequireAuth(pub)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, _ := Claims(r)
			for _, perm := range perms {
				if rbac.Allowed(claims.Role, perm) {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		}))
	}
}

func RequirePerm(pub *rsa.PublicKey, perm string) func(http.Handler) http.Handler {
	return RequireAnyPerm(pub, perm)
}
