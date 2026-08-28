package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type ChecklistRepository struct{}

func NewChecklistRepository() *ChecklistRepository {
	return &ChecklistRepository{}
}

func (r *ChecklistRepository) Create(cl *models.Checklist) error {
	return config.DB.Create(cl).Error
}

func (r *ChecklistRepository) FindByID(id uint) (*models.Checklist, error) {
	var cl models.Checklist
	err := config.DB.First(&cl, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cl, nil
}

// List mengambil daftar checklist dengan filter multi-tenant, action plan, dan task.
// perusahaanID = 0 -> semua perusahaan (Super Admin).
// actionPlanID = 0 -> tidak difilter action plan tertentu.
// taskID = 0 -> tidak difilter task tertentu.
func (r *ChecklistRepository) List(perusahaanID uint, actionPlanID uint, taskID uint) ([]models.Checklist, error) {
	var checklists []models.Checklist
	query := config.DB.Model(&models.Checklist{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	if actionPlanID > 0 {
		query = query.Where("action_plan_id = ?", actionPlanID)
	}
	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}
	err := query.Find(&checklists).Error
	return checklists, err
}

func (r *ChecklistRepository) Update(cl *models.Checklist) error {
	return config.DB.Save(cl).Error
}

func (r *ChecklistRepository) Delete(id uint) error {
	return config.DB.Delete(&models.Checklist{}, id).Error
}
