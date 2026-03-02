package middleware

import "testing"

func TestCSRFTokenRandom(t *testing.T) {
	a, _ := NewCSRFToken()
	b, _ := NewCSRFToken()
	if a == b || len(a) < 20 {
		t.Fatal("csrf token not random")
	}
}
