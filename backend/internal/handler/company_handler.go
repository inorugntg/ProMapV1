package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type CompanyHandler struct {
	companyRepo *repository.CompanyRepository
}

func NewCompanyHandler(companyRepo *repository.CompanyRepository) *CompanyHandler {
	return &CompanyHandler{companyRepo: companyRepo}
}

// ListCompanies mengembalikan daftar perusahaan. Super Admin melihat semua perusahaan,
// role lain hanya melihat perusahaannya sendiri (perusahaan_id/company_id dari JWT).
func (h *CompanyHandler) ListCompanies(c *gin.Context) {
	auth := getAuthContext(c)

	filterID := auth.PerusahaanID
	if auth.Role == utils.RoleSuperAdmin {
		filterID = 0
	}

	companies, err := h.companyRepo.List(filterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data perusahaan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"companies": companies})
}

type createCompanyRequest struct {
	Nama               string `json:"nama" binding:"required"`
	KodeUnik           string `json:"kode_unik" binding:"required"`
	StatusSubscription string `json:"status_subscription"`
}

// CreateCompany membuat perusahaan baru. Hanya Super Admin (route dibatasi RequireRole).
// status_subscription hanya boleh "basic" atau "premium"; jika kosong, default "basic"
// (mengikuti default kolom pada models.go).
func (h *CompanyHandler) CreateCompany(c *gin.Context) {
	var req createCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.StatusSubscription != "" && req.StatusSubscription != "basic" && req.StatusSubscription != "premium" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status_subscription harus 'basic' atau 'premium'"})
		return
	}

	exists, err := h.companyRepo.KodeUnikExists(req.KodeUnik)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi kode unik"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Kode unik sudah digunakan"})
		return
	}

	company := models.Perusahaan{
		Nama:               req.Nama,
		KodeUnik:           req.KodeUnik,
		StatusSubscription: req.StatusSubscription,
	}

	if err := h.companyRepo.Create(&company); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat perusahaan"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Perusahaan berhasil dibuat", "company": company})
}

type updateCompanyRequest struct {
	Nama               *string `json:"nama"`
	KodeUnik           *string `json:"kode_unik"`
	StatusSubscription *string `json:"status_subscription"`
}

// UpdateCompany mengubah data perusahaan. Hanya Super Admin (route dibatasi RequireRole).
func (h *CompanyHandler) UpdateCompany(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID perusahaan tidak valid"})
		return
	}

	company, err := h.companyRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data perusahaan"})
		return
	}
	if company == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Perusahaan tidak ditemukan"})
		return
	}

	var req updateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Nama != nil {
		company.Nama = *req.Nama
	}
	if req.KodeUnik != nil && *req.KodeUnik != company.KodeUnik {
		exists, err := h.companyRepo.KodeUnikExistsExcept(*req.KodeUnik, company.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi kode unik"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Kode unik sudah digunakan"})
			return
		}
		company.KodeUnik = *req.KodeUnik
	}
	if req.StatusSubscription != nil {
		if *req.StatusSubscription != "basic" && *req.StatusSubscription != "premium" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status_subscription harus 'basic' atau 'premium'"})
			return
		}
		company.StatusSubscription = *req.StatusSubscription
	}

	if err := h.companyRepo.Update(company); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui perusahaan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Perusahaan berhasil diperbarui", "company": company})
}

// DeleteCompany menghapus perusahaan (soft delete). Hanya Super Admin (route dibatasi RequireRole).
func (h *CompanyHandler) DeleteCompany(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID perusahaan tidak valid"})
		return
	}

	company, err := h.companyRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data perusahaan"})
		return
	}
	if company == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Perusahaan tidak ditemukan"})
		return
	}

	if err := h.companyRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus perusahaan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Perusahaan berhasil dihapus"})
}
