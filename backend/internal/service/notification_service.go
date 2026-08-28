package service

import (
	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type NotificationService struct{}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// Create menyimpan notifikasi in-app baru untuk seorang user.
func (s *NotificationService) Create(userID uint, title string, message string, notifType string) error {
	notification := models.Notification{
		UserID:  userID,
		Title:   title,
		Message: message,
		Type:    notifType,
		IsRead:  false,
	}
	return config.DB.Create(&notification).Error
}

// ListByUser mengambil seluruh notifikasi milik seorang user, terbaru lebih dulu.
func (s *NotificationService) ListByUser(userID uint) ([]models.Notification, error) {
	var notifications []models.Notification
	err := config.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

// MarkAllRead menandai seluruh notifikasi milik seorang user sebagai sudah dibaca.
func (s *NotificationService) MarkAllRead(userID uint) error {
	return config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

// MarkRead menandai satu notifikasi milik seorang user sebagai sudah dibaca.
// Mengembalikan gorm.ErrRecordNotFound jika notifikasi tidak ditemukan atau bukan milik user tsb.
func (s *NotificationService) MarkRead(id uint, userID uint) error {
	result := config.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
