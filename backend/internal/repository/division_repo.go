package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type DivisionRepository struct{}

func NewDivisionRepository() *DivisionRepository {
	return &DivisionRepository{}
}

func (r *DivisionRepository) Create(d *models.Divisi) error {
	return config.DB.Create(d).Error
}

func (r *DivisionRepository) FindByID(id uint) (*models.Divisi, error) {
	var d models.Divisi
	err := config.DB.First(&d, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// List mengambil daftar divisi. perusahaanID = 0 -> semua perusahaan (Super Admin).
// divisiID = 0 -> tidak dibatasi ke satu divisi tertentu; divisiID > 0 dipakai untuk
// membatasi role selain Super Admin/Admin Operasional agar hanya melihat divisinya sendiri.
func (r *DivisionRepository) List(perusahaanID uint, divisiID uint) ([]models.Divisi, error) {
	var divisions []models.Divisi
	query := config.DB.Model(&models.Divisi{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	if divisiID > 0 {
		query = query.Where("id = ?", divisiID)
	}
	err := query.Find(&divisions).Error
	return divisions, err
}

func (r *DivisionRepository) Update(d *models.Divisi) error {
	return config.DB.Save(d).Error
}

func (r *DivisionRepository) Delete(id uint) error {
	return config.DB.Delete(&models.Divisi{}, id).Error
}
