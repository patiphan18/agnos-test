package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Config struct {
	Secret   string
	Issuer   string
	Audience string
}

type Claims struct {
	HospitalID string `json:"hospital_id"`
	jwt.RegisteredClaims
}

func Issue(config Config, staffID, hospitalID string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{HospitalID: hospitalID, RegisteredClaims: jwt.RegisteredClaims{Subject: staffID, ID: uuid.NewString(), Issuer: config.Issuer, Audience: jwt.ClaimStrings{config.Audience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}}).SignedString([]byte(config.Secret))
}
func Parse(config Config, raw string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(config.Secret), nil
	}, jwt.WithIssuer(config.Issuer), jwt.WithAudience(config.Audience), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return Claims{}, fmt.Errorf("invalid token")
	}
	return claims, nil
}
