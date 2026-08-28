package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type DivisionHandler struct {
	divisionRepo *repository.DivisionRepository
	companyRepo  *repository.CompanyRepository
}

func NewDivisionHandler(divisionRepo *repository.DivisionRepository, companyRepo *repository.CompanyRepository) *DivisionHandler {
	return &DivisionHandler{divisionRepo: divisionRepo, companyRepo: companyRepo}
}

// ListDivisions mengembalikan daftar divisi sesuai lingkup akses role. Super Admin melihat
// semua divisi di semua perusahaan, Admin Operasional melihat seluruh divisi di
// perusahaannya, dan role lain (Manager dan seterusnya) hanya melihat divisinya sendiri
// (divisi_id dari JWT).
func (h *DivisionHandler) ListDivisions(c *gin.Context) {
	auth := getAuthContext(c)

	filterPerusahaanID := auth.PerusahaanID
	var filterDivisiID uint
	switch {
	case auth.Role == utils.RoleSuperAdmin:
		filterPerusahaanID = 0
	case auth.Role == utils.RoleAdminOperasional:
		// seluruh divisi di perusahaannya, tidak dibatasi ke satu divisi
	default:
		filterDivisiID = auth.DivisiID
	}

	divisions, err := h.divisionRepo.List(filterPerusahaanID, filterDivisiID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data divisi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"divisions": divisions})
}

type createDivisionRequest struct {
	Nama         string `json:"nama" binding:"required"`
	Deskripsi    string `json:"deskripsi"`
	PerusahaanID uint   `json:"perusahaan_id"`
}

// CreateDivision membuat divisi baru. Super Admin & Admin Operasional (route dibatasi
// RequireRole). Admin Operasional hanya boleh membuat divisi di perusahaannya sendiri
// (perusahaan_id dari body diabaikan demi keamanan multi-tenant); Super Admin bebas
// menentukan perusahaan_id target.
func (h *DivisionHandler) CreateDivision(c *gin.Context) {
	var req createDivisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auth := getAuthContext(c)
	targetPerusahaanID := auth.PerusahaanID
	if auth.Role == utils.RoleSuperAdmin {
		if req.PerusahaanID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "perusahaan_id wajib diisi"})
			return
		}
		targetPerusahaanID = req.PerusahaanID
	}

	company, err := h.companyRepo.FindByID(targetPerusahaanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi perusahaan"})
		return
	}
	if company == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Perusahaan tidak ditemukan"})
		return
	}

	division := models.Divisi{
		PerusahaanID: targetPerusahaanID,
		Nama:         req.Nama,
		Deskripsi:    req.Deskripsi,
	}

	if err := h.divisionRepo.Create(&division); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat divisi"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Divisi berhasil dibuat", "division": division})
}

type updateDivisionRequest struct {
	Nama      *string `json:"nama"`
	Deskripsi *string `json:"deskripsi"`
}

// UpdateDivision mengubah data divisi. Super Admin & Admin Operasional (route dibatasi
// RequireRole); Admin Operasional hanya boleh mengubah divisi di perusahaannya sendiri.
func (h *DivisionHandler) UpdateDivision(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID divisi tidak valid"})
		return
	}

	division, err := h.divisionRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data divisi"})
		return
	}
	if division == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Divisi tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && division.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola divisi di luar perusahaan Anda"})
		return
	}

	var req updateDivisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Nama != nil {
		division.Nama = *req.Nama
	}
	if req.Deskripsi != nil {
		division.Deskripsi = *req.Deskripsi
	}

	if err := h.divisionRepo.Update(division); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui divisi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Divisi berhasil diperbarui", "division": division})
}

// DeleteDivision menghapus divisi (soft delete). Super Admin & Admin Operasional (route
// dibatasi RequireRole); Admin Operasional hanya boleh menghapus divisi di perusahaannya sendiri.
func (h *DivisionHandler) DeleteDivision(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID divisi tidak valid"})
		return
	}

	division, err := h.divisionRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data divisi"})
		return
	}
	if division == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Divisi tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && division.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola divisi di luar perusahaan Anda"})
		return
	}

	if err := h.divisionRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus divisi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Divisi berhasil dihapus"})
}
