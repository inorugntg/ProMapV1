package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type ObjectiveRepository struct{}

func NewObjectiveRepository() *ObjectiveRepository {
	return &ObjectiveRepository{}
}

func (r *ObjectiveRepository) Create(o *models.Objective) error {
	return config.DB.Create(o).Error
}

func (r *ObjectiveRepository) FindByID(id uint) (*models.Objective, error) {
	var o models.Objective
	err := config.DB.First(&o, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// List mengambil daftar objective dengan filter multi-tenant dan opsional per project.
func (r *ObjectiveRepository) List(perusahaanID uint, divisiID *uint, projectID uint) ([]models.Objective, error) {
	var objectives []models.Objective
	query := config.DB.Model(&models.Objective{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	if divisiID != nil {
		query = query.Where("divisi_id = ?", *divisiID)
	}
	if projectID > 0 {
		query = query.Where("project_id = ?", projectID)
	}
	err := query.Find(&objectives).Error
	return objectives, err
}

func (r *ObjectiveRepository) Update(o *models.Objective) error {
	return config.DB.Save(o).Error
}

func (r *ObjectiveRepository) Delete(id uint) error {
	return config.DB.Delete(&models.Objective{}, id).Error
}
