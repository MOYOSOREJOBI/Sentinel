package auth

import (
	"crypto/rsa"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func ReadPrivate(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPrivateKeyFromPEM(b)
}
func ReadPublic(path string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPublicKeyFromPEM(b)
}

func Sign(userID, role string, key *rsa.PrivateKey) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{Role: role, RegisteredClaims: jwt.RegisteredClaims{Subject: userID, ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}})
	return t.SignedString(key)
}

func Parse(token string, pub *rsa.PublicKey) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return pub, nil
	})
	if err != nil || !parsed.Valid {
		if err == nil {
			err = jwt.ErrTokenInvalidClaims
		}
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || claims.Subject == "" || claims.Role == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
