package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type ActionPlanRepository struct{}

func NewActionPlanRepository() *ActionPlanRepository {
	return &ActionPlanRepository{}
}

func (r *ActionPlanRepository) Create(a *models.ActionPlan) error {
	return config.DB.Create(a).Error
}

func (r *ActionPlanRepository) FindByID(id uint) (*models.ActionPlan, error) {
	var a models.ActionPlan
	err := config.DB.First(&a, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// List mengambil daftar action plan dengan filter multi-tenant, divisi, pemilik, dan task.
// perusahaanID = 0 -> semua perusahaan (Super Admin).
// divisionID = 0 -> tidak difilter divisi.
// userID = 0 -> tidak difilter pemilik (dipakai Manager/Admin/SuperAdmin).
// taskID = 0 -> tidak difilter task tertentu.
func (r *ActionPlanRepository) List(perusahaanID uint, divisionID uint, userID uint, taskID uint) ([]models.ActionPlan, error) {
	var plans []models.ActionPlan
	query := config.DB.Model(&models.ActionPlan{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	if divisionID > 0 {
		query = query.Where("division_id = ?", divisionID)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}
	err := query.Find(&plans).Error
	return plans, err
}

func (r *ActionPlanRepository) Update(a *models.ActionPlan) error {
	return config.DB.Save(a).Error
}

func (r *ActionPlanRepository) Delete(id uint) error {
	return config.DB.Delete(&models.ActionPlan{}, id).Error
}
