package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type ProposalRepository struct{}

func NewProposalRepository() *ProposalRepository {
	return &ProposalRepository{}
}

func (r *ProposalRepository) Create(p *models.Proposal) error {
	return config.DB.Create(p).Error
}

func (r *ProposalRepository) FindByID(id uint) (*models.Proposal, error) {
	var p models.Proposal
	err := config.DB.First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List mengambil daftar proposal dengan filter multi-tenant, divisi, pengaju, dan status.
// perusahaanID = 0 -> semua perusahaan (Super Admin).
// divisiID = nil -> tidak difilter divisi.
// createdByID = 0 -> tidak difilter pengaju (dipakai Manager/Admin/SuperAdmin).
// status = "" -> semua status.
func (r *ProposalRepository) List(perusahaanID uint, divisiID *uint, createdByID uint, status string) ([]models.Proposal, error) {
	var proposals []models.Proposal
	query := config.DB.Model(&models.Proposal{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	if divisiID != nil {
		query = query.Where("divisi_id = ?", *divisiID)
	}
	if createdByID > 0 {
		query = query.Where("created_by_id = ?", createdByID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&proposals).Error
	return proposals, err
}

func (r *ProposalRepository) Update(p *models.Proposal) error {
	return config.DB.Save(p).Error
}

func (r *ProposalRepository) Delete(id uint) error {
	return config.DB.Delete(&models.Proposal{}, id).Error
}
