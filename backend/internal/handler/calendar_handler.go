package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/service"
)

type CalendarHandler struct {
	calendarService *service.CalendarService
}

func NewCalendarHandler(calendarService *service.CalendarService) *CalendarHandler {
	return &CalendarHandler{calendarService: calendarService}
}

// GetEvents mengembalikan Task & Action Plan (start_date/end_date) dalam lingkup akses role,
// untuk ditampilkan sebagai kalender.
func (h *CalendarHandler) GetEvents(c *gin.Context) {
	auth := getAuthContext(c)

	events, err := h.calendarService.GetEvents(auth.Role, auth.PerusahaanID, auth.DivisiID, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kalender"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}
