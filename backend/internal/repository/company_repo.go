package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type CompanyRepository struct{}

func NewCompanyRepository() *CompanyRepository {
	return &CompanyRepository{}
}

func (r *CompanyRepository) Create(p *models.Perusahaan) error {
	return config.DB.Create(p).Error
}

func (r *CompanyRepository) FindByID(id uint) (*models.Perusahaan, error) {
	var p models.Perusahaan
	err := config.DB.First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List mengambil daftar perusahaan. id = 0 -> semua perusahaan (khusus Super Admin).
// id > 0 -> hanya perusahaan dengan id tersebut (dipakai role selain Super Admin,
// yang hanya boleh melihat perusahaannya sendiri berdasarkan company_id dari JWT).
func (r *CompanyRepository) List(id uint) ([]models.Perusahaan, error) {
	var companies []models.Perusahaan
	query := config.DB.Model(&models.Perusahaan{})
	if id > 0 {
		query = query.Where("id = ?", id)
	}
	err := query.Find(&companies).Error
	return companies, err
}

func (r *CompanyRepository) Update(p *models.Perusahaan) error {
	return config.DB.Save(p).Error
}

func (r *CompanyRepository) Delete(id uint) error {
	return config.DB.Delete(&models.Perusahaan{}, id).Error
}

// KodeUnikExists mengecek apakah kode_unik sudah dipakai perusahaan lain.
func (r *CompanyRepository) KodeUnikExists(kodeUnik string) (bool, error) {
	var count int64
	err := config.DB.Model(&models.Perusahaan{}).Where("kode_unik = ?", kodeUnik).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// KodeUnikExistsExcept mengecek apakah kode_unik sudah dipakai perusahaan lain,
// selain perusahaan dengan id excludeID (dipakai saat validasi update).
func (r *CompanyRepository) KodeUnikExistsExcept(kodeUnik string, excludeID uint) (bool, error) {
	var count int64
	err := config.DB.Model(&models.Perusahaan{}).Where("kode_unik = ? AND id <> ?", kodeUnik, excludeID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
