package handler

import (
	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/middleware"
)

// authContext merangkum identitas user yang sedang login (hasil parsing JWT)
// supaya tidak perlu memanggil c.MustGet berulang-ulang di tiap handler.
type authContext struct {
	UserID       uint
	Role         string
	PerusahaanID uint
	DivisiID     uint
}

func getAuthContext(c *gin.Context) authContext {
	return authContext{
		UserID:       c.MustGet(middleware.ContextUserID).(uint),
		Role:         c.MustGet(middleware.ContextRole).(string),
		PerusahaanID: c.MustGet(middleware.ContextPerusahaanID).(uint),
		DivisiID:     c.MustGet(middleware.ContextDivisiID).(uint),
	}
}
