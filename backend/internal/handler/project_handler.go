package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type ProjectHandler struct {
	projectRepo *repository.ProjectRepository
	userRepo    *repository.UserRepository
}

func NewProjectHandler(projectRepo *repository.ProjectRepository, userRepo *repository.UserRepository) *ProjectHandler {
	return &ProjectHandler{projectRepo: projectRepo, userRepo: userRepo}
}

// ListProjects mengembalikan daftar project sesuai lingkup akses role:
// Super Admin melihat semua perusahaan, Admin Operasional melihat seluruh divisi
// di perusahaannya, Manager/PIC/Staff hanya melihat project pada divisinya sendiri.
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	auth := getAuthContext(c)

	filterPerusahaanID := auth.PerusahaanID
	var filterDivisiID *uint
	if auth.Role == utils.RoleSuperAdmin {
		filterPerusahaanID = 0
	} else if !utils.IsManagementRole(auth.Role) || auth.Role == utils.RoleManager {
		filterDivisiID = &auth.DivisiID
	}

	projects, err := h.projectRepo.List(filterPerusahaanID, filterDivisiID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data project"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

type createProjectRequest struct {
	Nama         string `json:"nama" binding:"required"`
	Deskripsi    string `json:"deskripsi"`
	Status       string `json:"status"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	DivisiID     *uint  `json:"divisi_id"`
	PerusahaanID uint   `json:"perusahaan_id"`
}

// CreateProject membuat project baru. Hanya Super Admin, Admin Operasional, dan Manager
// yang boleh membuat project (route sudah dibatasi RequireRole).
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req createProjectRequest
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

	if req.DivisiID != nil {
		valid, err := h.userRepo.DivisiBelongsToPerusahaan(*req.DivisiID, targetPerusahaanID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi divisi"})
			return
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Divisi tidak ditemukan pada perusahaan ini"})
			return
		}
	} else if auth.Role == utils.RoleManager {
		req.DivisiID = &auth.DivisiID
	}

	status := req.Status
	if status == "" {
		status = "Not Started"
	}

	project := models.Project{
		PerusahaanID: targetPerusahaanID,
		DivisiID:     req.DivisiID,
		UserID:       auth.UserID,
		Nama:         req.Nama,
		Deskripsi:    req.Deskripsi,
		Status:       status,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
	}

	if err := h.projectRepo.Create(&project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat project"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Project berhasil dibuat", "project": project})
}

type updateProjectRequest struct {
	Nama      *string `json:"nama"`
	Deskripsi *string `json:"deskripsi"`
	Status    *string `json:"status"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	DivisiID  *uint   `json:"divisi_id"`
}

// UpdateProject mengedit project. Manager hanya boleh mengedit project di divisinya sendiri;
// Admin Operasional/Super Admin boleh mengedit project mana pun di lingkupnya.
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID project tidak valid"})
		return
	}

	project, err := h.projectRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data project"})
		return
	}
	if project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && project.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola project di luar perusahaan Anda"})
		return
	}
	if auth.Role == utils.RoleManager && (project.DivisiID == nil || *project.DivisiID != auth.DivisiID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola project di luar divisi Anda"})
		return
	}

	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Nama != nil {
		project.Nama = *req.Nama
	}
	if req.Deskripsi != nil {
		project.Deskripsi = *req.Deskripsi
	}
	if req.Status != nil {
		project.Status = *req.Status
	}
	if req.StartDate != nil {
		project.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		project.EndDate = *req.EndDate
	}
	if req.DivisiID != nil {
		valid, err := h.userRepo.DivisiBelongsToPerusahaan(*req.DivisiID, project.PerusahaanID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi divisi"})
			return
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Divisi tidak ditemukan pada perusahaan ini"})
			return
		}
		project.DivisiID = req.DivisiID
	}

	if err := h.projectRepo.Update(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui project"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Project berhasil diperbarui", "project": project})
}

// DeleteProject menghapus project (soft delete). Hanya Super Admin dan Admin Operasional.
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID project tidak valid"})
		return
	}

	project, err := h.projectRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data project"})
		return
	}
	if project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && project.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola project di luar perusahaan Anda"})
		return
	}

	if err := h.projectRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus project"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Project berhasil dihapus"})
}
