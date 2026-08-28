package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type EvidenceRepository struct{}

func NewEvidenceRepository() *EvidenceRepository {
	return &EvidenceRepository{}
}

func (r *EvidenceRepository) Create(e *models.Evidence) error {
	return config.DB.Create(e).Error
}

func (r *EvidenceRepository) FindByID(id uint) (*models.Evidence, error) {
	var e models.Evidence
	err := config.DB.First(&e, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// List mengambil daftar evidence dengan filter multi-tenant, pemilik, action plan tertentu,
// dan (opsional) daftar action_plan_id yang diizinkan -- dipakai untuk membatasi Manager
// hanya melihat evidence pada action plan di divisinya sendiri.
// perusahaanID = 0 -> semua perusahaan (Super Admin).
// userID = 0 -> tidak difilter pemilik (dipakai Manager/Admin/SuperAdmin).
// actionPlanID = 0 -> tidak difilter ke satu action plan tertentu.
// allowedActionPlanIDs = nil -> tidak dibatasi; non-nil (termasuk slice kosong) -> hanya
// evidence dengan action_plan_id di dalam daftar tersebut.
func (r *EvidenceRepository) List(perusahaanID uint, userID uint, actionPlanID uint, allowedActionPlanIDs []uint) ([]models.Evidence, error) {
	var evidences []models.Evidence
	query := config.DB.Model(&models.Evidence{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if actionPlanID > 0 {
		query = query.Where("action_plan_id = ?", actionPlanID)
	}
	if allowedActionPlanIDs != nil {
		query = query.Where("action_plan_id IN ?", allowedActionPlanIDs)
	}
	err := query.Find(&evidences).Error
	return evidences, err
}

func (r *EvidenceRepository) Update(e *models.Evidence) error {
	return config.DB.Save(e).Error
}

func (r *EvidenceRepository) Delete(id uint) error {
	return config.DB.Delete(&models.Evidence{}, id).Error
}
