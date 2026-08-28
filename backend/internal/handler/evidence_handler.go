package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type EvidenceHandler struct {
	evidenceRepo   *repository.EvidenceRepository
	actionPlanRepo *repository.ActionPlanRepository
}

func NewEvidenceHandler(evidenceRepo *repository.EvidenceRepository, actionPlanRepo *repository.ActionPlanRepository) *EvidenceHandler {
	return &EvidenceHandler{evidenceRepo: evidenceRepo, actionPlanRepo: actionPlanRepo}
}

// ListEvidences mengembalikan daftar evidence sesuai lingkup akses role. Super Admin:
// semua perusahaan. Admin Operasional: seluruh divisi di perusahaannya. Manager: evidence
// pada action plan di divisinya sendiri. PIC/Staff: hanya evidence miliknya.
// Query param opsional ?action_plan_id= untuk memfilter evidence pada satu action plan.
func (h *EvidenceHandler) ListEvidences(c *gin.Context) {
	auth := getAuthContext(c)

	filterPerusahaanID := auth.PerusahaanID
	var filterUserID uint
	var allowedActionPlanIDs []uint

	switch {
	case auth.Role == utils.RoleSuperAdmin:
		filterPerusahaanID = 0
	case auth.Role == utils.RoleAdminOperasional:
		// seluruh divisi di perusahaannya
	case auth.Role == utils.RoleManager:
		plans, err := h.actionPlanRepo.List(auth.PerusahaanID, auth.DivisiID, 0, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data action plan"})
			return
		}
		allowedActionPlanIDs = make([]uint, len(plans))
		for i, p := range plans {
			allowedActionPlanIDs[i] = p.ID
		}
	default: // PIC / Staff
		filterUserID = auth.UserID
	}

	var actionPlanID uint
	if v := c.Query("action_plan_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "action_plan_id tidak valid"})
			return
		}
		actionPlanID = uint(id)
	}

	evidences, err := h.evidenceRepo.List(filterPerusahaanID, filterUserID, actionPlanID, allowedActionPlanIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data evidence"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"evidences": evidences})
}

type createEvidenceRequest struct {
	ActionPlanID uint   `json:"action_plan_id" binding:"required"`
	Judul        string `json:"judul" binding:"required"`
	Deskripsi    string `json:"deskripsi"`
	Link         string `json:"link"`
}

// CreateEvidence membuat evidence baru untuk sebuah action plan, oleh PIC/Staff pemilik
// action plan tersebut, atau oleh Manager/Admin/Super Admin dalam lingkupnya. Status awal
// selalu "Pending" menunggu approve/reject dari Manager/Admin.
func (h *EvidenceHandler) CreateEvidence(c *gin.Context) {
	var req createEvidenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plan, err := h.actionPlanRepo.FindByID(req.ActionPlanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi action plan"})
		return
	}
	if plan == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action plan tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && plan.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat membuat evidence di luar perusahaan Anda"})
		return
	}
	if utils.IsExecutorRole(auth.Role) && plan.UserID != auth.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat membuat evidence untuk action plan milik Anda sendiri"})
		return
	}
	if auth.Role == utils.RoleManager && plan.DivisionID != auth.DivisiID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat membuat evidence di luar divisi Anda"})
		return
	}

	evidence := models.Evidence{
		ActionPlanID: plan.ID,
		TaskID:       plan.TaskID,
		PerusahaanID: plan.PerusahaanID,
		UserID:       auth.UserID,
		Judul:        req.Judul,
		Deskripsi:    req.Deskripsi,
		Link:         req.Link,
		Status:       "Pending",
	}

	if err := h.evidenceRepo.Create(&evidence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat evidence"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Evidence berhasil dibuat", "evidence": evidence})
}

type updateEvidenceRequest struct {
	Judul        *string `json:"judul"`
	Deskripsi    *string `json:"deskripsi"`
	Link         *string `json:"link"`
	Status       *string `json:"status" binding:"omitempty,oneof=Approved Rejected"`
	CatatanAdmin *string `json:"catatan_admin"`
}

// UpdateEvidence mengedit evidence. Pemilik (PIC/Staff) hanya boleh mengubah isi evidence
// (judul/deskripsi/link) selagi status masih "Pending". Manager/Admin/Super Admin dalam
// lingkupnya hanya boleh mengubah status (approve/reject) dan catatan_admin.
func (h *EvidenceHandler) UpdateEvidence(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID evidence tidak valid"})
		return
	}

	evidence, err := h.evidenceRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data evidence"})
		return
	}
	if evidence == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Evidence tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	var req updateEvidenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if utils.IsExecutorRole(auth.Role) {
		if evidence.UserID != auth.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat mengubah evidence milik Anda sendiri"})
			return
		}
		if evidence.Status != "Pending" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Evidence yang sudah diproses tidak dapat diubah isinya"})
			return
		}
		if req.Judul != nil {
			evidence.Judul = *req.Judul
		}
		if req.Deskripsi != nil {
			evidence.Deskripsi = *req.Deskripsi
		}
		if req.Link != nil {
			evidence.Link = *req.Link
		}
	} else {
		if auth.Role != utils.RoleSuperAdmin && evidence.PerusahaanID != auth.PerusahaanID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola evidence di luar perusahaan Anda"})
			return
		}
		if auth.Role == utils.RoleManager {
			plan, err := h.actionPlanRepo.FindByID(evidence.ActionPlanID)
			if err != nil || plan == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi action plan induk"})
				return
			}
			if plan.DivisionID != auth.DivisiID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola evidence di luar divisi Anda"})
				return
			}
		}

		if req.Status != nil {
			evidence.Status = *req.Status
		}
		if req.CatatanAdmin != nil {
			evidence.CatatanAdmin = *req.CatatanAdmin
		}
	}

	if err := h.evidenceRepo.Update(evidence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui evidence"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Evidence berhasil diperbarui", "evidence": evidence})
}

// DeleteEvidence menghapus evidence (soft delete). Hanya pemilik evidence, dan hanya
// selagi status masih "Pending" -- Manager/Admin/Super Admin tidak dapat menghapus evidence
// milik orang lain, sesuai spesifikasi.
func (h *EvidenceHandler) DeleteEvidence(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID evidence tidak valid"})
		return
	}

	evidence, err := h.evidenceRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data evidence"})
		return
	}
	if evidence == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Evidence tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if evidence.UserID != auth.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat menghapus evidence milik Anda sendiri"})
		return
	}
	if evidence.Status != "Pending" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Evidence yang sudah diproses tidak dapat dihapus"})
		return
	}

	if err := h.evidenceRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus evidence"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Evidence berhasil dihapus"})
}
