package repository

import (
	"errors"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// FindByEmail mencari user berdasarkan email (dipakai saat login & cek duplikasi)
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := config.DB.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID mencari user berdasarkan primary key
func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	err := config.DB.First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// EmailExists mengecek apakah email sudah dipakai user lain
func (r *UserRepository) EmailExists(email string) (bool, error) {
	var count int64
	err := config.DB.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// PerusahaanExists mengecek apakah perusahaan_id merujuk ke perusahaan yang benar-benar ada
func (r *UserRepository) PerusahaanExists(id uint) (bool, error) {
	var count int64
	err := config.DB.Model(&models.Perusahaan{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DivisiBelongsToPerusahaan mengecek apakah divisi_id ada dan merupakan milik perusahaan yang dituju,
// mencegah user ditempatkan ke divisi milik perusahaan lain.
func (r *UserRepository) DivisiBelongsToPerusahaan(divisiID uint, perusahaanID uint) (bool, error) {
	var count int64
	err := config.DB.Model(&models.Divisi{}).Where("id = ? AND perusahaan_id = ?", divisiID, perusahaanID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Create menyimpan user baru
func (r *UserRepository) Create(user *models.User) error {
	return config.DB.Create(user).Error
}

// Update menyimpan perubahan pada user yang sudah ada
func (r *UserRepository) Update(user *models.User) error {
	return config.DB.Save(user).Error
}

// Delete melakukan soft delete (memanfaatkan gorm.Model.DeletedAt)
func (r *UserRepository) Delete(id uint) error {
	return config.DB.Delete(&models.User{}, id).Error
}

// List mengambil daftar user. Jika perusahaanID > 0, hasil difilter ke perusahaan tersebut
// (dipakai untuk role selain Super Admin, yang hanya boleh melihat user di perusahaannya sendiri).
func (r *UserRepository) List(perusahaanID uint) ([]models.User, error) {
	var users []models.User
	query := config.DB.Model(&models.User{})
	if perusahaanID > 0 {
		query = query.Where("perusahaan_id = ?", perusahaanID)
	}
	err := query.Find(&users).Error
	return users, err
}
