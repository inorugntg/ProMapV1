package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims adalah payload custom yang disisipkan ke dalam JWT
type Claims struct {
	UserID       uint   `json:"user_id"`
	Role         string `json:"role"`
	PerusahaanID uint   `json:"perusahaan_id"`
	DivisiID     uint   `json:"divisi_id"`
	jwt.RegisteredClaims
}

func jwtSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

// GenerateToken membuat JWT baru untuk user yang berhasil login
func GenerateToken(userID uint, role string, perusahaanID uint, divisiID uint) (string, error) {
	claims := Claims{
		UserID:       userID,
		Role:         role,
		PerusahaanID: perusahaanID,
		DivisiID:     divisiID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// ParseToken memvalidasi JWT dan mengembalikan claims di dalamnya
func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing token tidak valid")
		}
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token tidak valid")
	}

	return claims, nil
}
