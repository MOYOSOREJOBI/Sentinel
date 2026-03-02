package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseRejectsWrongAlgorithm(t *testing.T) {
	pubPriv, _ := rsa.GenerateKey(rand.Reader, 2048)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "a", "role": "admin", "exp": time.Now().Add(time.Hour).Unix()})
	s, _ := tok.SignedString([]byte("secret"))
	if _, err := Parse(s, &pubPriv.PublicKey); err == nil {
		t.Fatal("expected parse failure for wrong alg")
	}
}

func TestParseRejectsMissingRequiredClaims(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}})
	s, _ := tok.SignedString(key)
	if _, err := Parse(s, &key.PublicKey); err == nil {
		t.Fatal("expected parse failure for missing subject/role")
	}
}
