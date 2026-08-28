package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"corporate-action-plan/backend/internal/service"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
}

func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

// ListNotifications mengembalikan seluruh notifikasi milik user yang sedang login.
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	auth := getAuthContext(c)

	notifications, err := h.notificationService.ListByUser(auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data notifikasi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

// MarkAllRead menandai seluruh notifikasi milik user yang sedang login sebagai sudah dibaca.
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	auth := getAuthContext(c)

	if err := h.notificationService.MarkAllRead(auth.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menandai notifikasi sebagai sudah dibaca"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Semua notifikasi berhasil ditandai sebagai sudah dibaca"})
}

// MarkRead menandai satu notifikasi milik user yang sedang login sebagai sudah dibaca.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	auth := getAuthContext(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID notifikasi tidak valid"})
		return
	}

	if err := h.notificationService.MarkRead(uint(id), auth.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notifikasi tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menandai notifikasi sebagai sudah dibaca"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notifikasi berhasil ditandai sebagai sudah dibaca"})
}
