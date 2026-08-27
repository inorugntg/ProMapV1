package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/utils"
)

// Key yang dipakai untuk menyimpan info user hasil parsing JWT ke dalam gin.Context
const (
	ContextUserID       = "user_id"
	ContextRole         = "role"
	ContextPerusahaanID = "perusahaan_id"
	ContextDivisiID     = "divisi_id"
)

// AuthRequired memvalidasi header "Authorization: Bearer <token>" dan menyimpan
// identitas user (id, role, perusahaan_id, divisi_id) ke context untuk dipakai handler berikutnya.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token otorisasi tidak ditemukan"})
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid atau sudah kadaluarsa"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextRole, claims.Role)
		c.Set(ContextPerusahaanID, claims.PerusahaanID)
		c.Set(ContextDivisiID, claims.DivisiID)
		c.Next()
	}
}

// RequireRole membatasi akses endpoint hanya untuk role yang diizinkan.
// Harus dipasang setelah AuthRequired().
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		for _, allowed := range roles {
			if allowed == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses untuk melakukan aksi ini"})
	}
}
