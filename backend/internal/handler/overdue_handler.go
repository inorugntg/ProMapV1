package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/service"
)

type OverdueHandler struct {
	overdueService *service.OverdueService
}

func NewOverdueHandler(overdueService *service.OverdueService) *OverdueHandler {
	return &OverdueHandler{overdueService: overdueService}
}

// CheckOverdue memicu pengecekan status overdue Task & Action Plan secara manual
// (dipakai untuk testing; secara normal pengecekan berjalan otomatis via background worker).
func (h *OverdueHandler) CheckOverdue(c *gin.Context) {
	overdueTasks, overdueActionPlans, err := h.overdueService.CheckAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menjalankan pengecekan overdue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":              "Pengecekan overdue berhasil dijalankan",
		"overdue_tasks":        overdueTasks,
		"overdue_action_plans": overdueActionPlans,
	})
}
