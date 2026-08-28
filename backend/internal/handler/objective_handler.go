package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

// ObjectiveHandler mengelola Objective (turunan Project, induk dari Task). Tabel ini
// tidak diminta secara eksplisit di daftar endpoint, namun tetap dibutuhkan agar hierarki
// Project -> Objective -> Task -> ActionPlan -> Checklist dapat benar-benar dibuat/dipakai.
type ObjectiveHandler struct {
	objectiveRepo *repository.ObjectiveRepository
	projectRepo   *repository.ProjectRepository
	userRepo      *repository.UserRepository
}

func NewObjectiveHandler(objectiveRepo *repository.ObjectiveRepository, projectRepo *repository.ProjectRepository, userRepo *repository.UserRepository) *ObjectiveHandler {
	return &ObjectiveHandler{objectiveRepo: objectiveRepo, projectRepo: projectRepo, userRepo: userRepo}
}

// ListObjectives mengembalikan daftar objective sesuai lingkup akses role, opsional
// difilter dengan query param ?project_id=.
func (h *ObjectiveHandler) ListObjectives(c *gin.Context) {
	auth := getAuthContext(c)

	filterPerusahaanID := auth.PerusahaanID
	var filterDivisiID *uint
	if auth.Role == utils.RoleSuperAdmin {
		filterPerusahaanID = 0
	} else if !utils.IsManagementRole(auth.Role) || auth.Role == utils.RoleManager {
		filterDivisiID = &auth.DivisiID
	}

	var projectID uint
	if v := c.Query("project_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "project_id tidak valid"})
			return
		}
		projectID = uint(id)
	}

	objectives, err := h.objectiveRepo.List(filterPerusahaanID, filterDivisiID, projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data objective"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"objectives": objectives})
}

type createObjectiveRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Judul     string `json:"judul" binding:"required"`
	Deskripsi string `json:"deskripsi"`
	Status    string `json:"status"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// CreateObjective membuat objective baru di bawah sebuah project. Hanya Super Admin,
// Admin Operasional, dan Manager (route dibatasi RequireRole).
func (h *ObjectiveHandler) CreateObjective(c *gin.Context) {
	var req createObjectiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, err := h.projectRepo.FindByID(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi project"})
		return
	}
	if project == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project tidak ditemukan"})
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

	status := req.Status
	if status == "" {
		status = "Not Started"
	}

	objective := models.Objective{
		ProjectID:    project.ID,
		PerusahaanID: project.PerusahaanID,
		DivisiID:     project.DivisiID,
		Judul:        req.Judul,
		Deskripsi:    req.Deskripsi,
		Status:       status,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
	}

	if err := h.objectiveRepo.Create(&objective); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat objective"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Objective berhasil dibuat", "objective": objective})
}

type updateObjectiveRequest struct {
	Judul     *string `json:"judul"`
	Deskripsi *string `json:"deskripsi"`
	Status    *string `json:"status"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

// UpdateObjective mengedit objective. Lingkup akses sama seperti UpdateProject.
func (h *ObjectiveHandler) UpdateObjective(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID objective tidak valid"})
		return
	}

	objective, err := h.objectiveRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data objective"})
		return
	}
	if objective == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Objective tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && objective.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola objective di luar perusahaan Anda"})
		return
	}
	if auth.Role == utils.RoleManager && (objective.DivisiID == nil || *objective.DivisiID != auth.DivisiID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola objective di luar divisi Anda"})
		return
	}

	var req updateObjectiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Judul != nil {
		objective.Judul = *req.Judul
	}
	if req.Deskripsi != nil {
		objective.Deskripsi = *req.Deskripsi
	}
	if req.Status != nil {
		objective.Status = *req.Status
	}
	if req.StartDate != nil {
		objective.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		objective.EndDate = *req.EndDate
	}

	if err := h.objectiveRepo.Update(objective); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui objective"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Objective berhasil diperbarui", "objective": objective})
}

// DeleteObjective menghapus objective (soft delete). Hanya Super Admin dan Admin Operasional.
func (h *ObjectiveHandler) DeleteObjective(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID objective tidak valid"})
		return
	}

	objective, err := h.objectiveRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data objective"})
		return
	}
	if objective == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Objective tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && objective.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola objective di luar perusahaan Anda"})
		return
	}

	if err := h.objectiveRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus objective"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Objective berhasil dihapus"})
}
