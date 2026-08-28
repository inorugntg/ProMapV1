package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type ProjectRepository struct{}

func NewProjectRepository() *ProjectRepository {
	return &ProjectRepository{}
}

func (r *ProjectRepository) Create(p *models.Project) error {
	return config.DB.Create(p).Error
}

func (r *ProjectRepository) FindByID(id uint) (*models.Project, error) {
	var p models.Project
	err := config.DB.First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List mengambil daftar project dengan filter multi-tenant.
// perusahaanID = 0 berarti tidak difilter (khusus Super Admin).
// divisiID = nil berarti tidak difilter berdasarkan divisi.
func (r *ProjectRepository) List(perusahaanID uint, divisiID *uint) ([]models.Project, error) {
	var projects []models.Project
	query := config.DB.Model(&models.Project{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	if divisiID != nil {
		query = query.Where("divisi_id = ?", *divisiID)
	}
	err := query.Find(&projects).Error
	return projects, err
}

func (r *ProjectRepository) Update(p *models.Project) error {
	return config.DB.Save(p).Error
}

func (r *ProjectRepository) Delete(id uint) error {
	return config.DB.Delete(&models.Project{}, id).Error
}
