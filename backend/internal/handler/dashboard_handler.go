package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/service"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetDashboard mengembalikan statistik dashboard (total task/action plan, distribusi status
// & priority, progress per PIC, completion rate) sesuai lingkup akses role user yang login.
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	auth := getAuthContext(c)

	stats, err := h.dashboardService.GetDashboardStats(auth.UserID, auth.Role, auth.PerusahaanID, auth.DivisiID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data dashboard"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
