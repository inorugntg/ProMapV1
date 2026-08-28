package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type TaskRepository struct{}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (r *TaskRepository) Create(t *models.Task) error {
	return config.DB.Create(t).Error
}

func (r *TaskRepository) FindByID(id uint) (*models.Task, error) {
	var t models.Task
	err := config.DB.First(&t, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// List mengambil daftar task dengan filter multi-tenant, divisi, assignee, dan tipe.
// perusahaanID = 0 -> semua perusahaan (Super Admin).
// divisiID = nil -> tidak difilter divisi.
// assigneeID = 0 -> tidak difilter assignee (dipakai Manager/Admin/SuperAdmin).
// taskType = "" -> semua tipe (Task & PersonalTask).
func (r *TaskRepository) List(perusahaanID uint, divisiID *uint, assigneeID uint, taskType string) ([]models.Task, error) {
	var tasks []models.Task
	query := config.DB.Model(&models.Task{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	if divisiID != nil {
		query = query.Where("divisi_id = ?", *divisiID)
	}
	if assigneeID > 0 {
		query = query.Where("assignee_id = ?", assigneeID)
	}
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	err := query.Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepository) Update(t *models.Task) error {
	return config.DB.Save(t).Error
}

func (r *TaskRepository) Delete(id uint) error {
	return config.DB.Delete(&models.Task{}, id).Error
}
