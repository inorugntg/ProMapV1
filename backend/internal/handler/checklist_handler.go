package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type ChecklistHandler struct {
	checklistRepo  *repository.ChecklistRepository
	actionPlanRepo *repository.ActionPlanRepository
}

func NewChecklistHandler(checklistRepo *repository.ChecklistRepository, actionPlanRepo *repository.ActionPlanRepository) *ChecklistHandler {
	return &ChecklistHandler{checklistRepo: checklistRepo, actionPlanRepo: actionPlanRepo}
}

// authorizeActionPlanAccess memastikan user berhak mengakses action plan tertentu
// (pemilik action plan, atau Manager/Admin/Super Admin dalam lingkupnya).
func (h *ChecklistHandler) authorizeActionPlanAccess(auth authContext, plan *models.ActionPlan) bool {
	if utils.IsExecutorRole(auth.Role) {
		return plan.UserID == auth.UserID
	}
	if auth.Role != utils.RoleSuperAdmin && plan.PerusahaanID != auth.PerusahaanID {
		return false
	}
	if auth.Role == utils.RoleManager && (plan.DivisiID == nil || *plan.DivisiID != auth.DivisiID) {
		return false
	}
	return true
}

// ListChecklists mengembalikan daftar checklist. Wajib menyertakan query param
// ?action_plan_id= agar kepemilikan action plan induknya dapat divalidasi; opsional
// ?task_id= untuk mempersempit hasil pada tingkat task.
func (h *ChecklistHandler) ListChecklists(c *gin.Context) {
	auth := getAuthContext(c)

	var actionPlanID uint
	if v := c.Query("action_plan_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "action_plan_id tidak valid"})
			return
		}
		actionPlanID = uint(id)
	}

	filterPerusahaanID := auth.PerusahaanID
	if auth.Role == utils.RoleSuperAdmin {
		filterPerusahaanID = 0
	}

	if actionPlanID > 0 {
		plan, err := h.actionPlanRepo.FindByID(actionPlanID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi action plan"})
			return
		}
		if plan == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Action plan tidak ditemukan"})
			return
		}
		if !h.authorizeActionPlanAccess(auth, plan) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengakses checklist action plan ini"})
			return
		}
	} else if utils.IsExecutorRole(auth.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action_plan_id wajib diisi"})
		return
	}

	var taskID uint
	if v := c.Query("task_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task_id tidak valid"})
			return
		}
		taskID = uint(id)
	}

	checklists, err := h.checklistRepo.List(filterPerusahaanID, actionPlanID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data checklist"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"checklists": checklists})
}

type createChecklistRequest struct {
	ActionPlanID uint   `json:"action_plan_id" binding:"required"`
	NamaItem     string `json:"nama_item" binding:"required"`
	Status       string `json:"status"`
}

// CreateChecklist membuat item checklist baru di bawah sebuah action plan, oleh pemilik
// action plan tersebut (atau Manager/Admin/Super Admin dalam lingkupnya).
func (h *ChecklistHandler) CreateChecklist(c *gin.Context) {
	var req createChecklistRequest
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
	if !h.authorizeActionPlanAccess(auth, plan) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola checklist action plan ini"})
		return
	}

	status := req.Status
	if status == "" {
		status = "Not Done"
	}

	checklist := models.Checklist{
		ActionPlanID: plan.ID,
		TaskID:       plan.TaskID,
		PerusahaanID: plan.PerusahaanID,
		NamaItem:     req.NamaItem,
		Status:       status,
	}

	if err := h.checklistRepo.Create(&checklist); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat checklist"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Checklist berhasil dibuat", "checklist": checklist})
}

type updateChecklistRequest struct {
	NamaItem *string `json:"nama_item"`
	Status   *string `json:"status"`
}

// UpdateChecklist mengedit nama item atau status (Done/Not Done) sebuah checklist.
func (h *ChecklistHandler) UpdateChecklist(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID checklist tidak valid"})
		return
	}

	checklist, err := h.checklistRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data checklist"})
		return
	}
	if checklist == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Checklist tidak ditemukan"})
		return
	}

	plan, err := h.actionPlanRepo.FindByID(checklist.ActionPlanID)
	if err != nil || plan == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi action plan induk"})
		return
	}

	auth := getAuthContext(c)
	if !h.authorizeActionPlanAccess(auth, plan) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola checklist action plan ini"})
		return
	}

	var req updateChecklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NamaItem != nil {
		checklist.NamaItem = *req.NamaItem
	}
	if req.Status != nil {
		checklist.Status = *req.Status
	}

	if err := h.checklistRepo.Update(checklist); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui checklist"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Checklist berhasil diperbarui", "checklist": checklist})
}

// DeleteChecklist menghapus item checklist (soft delete).
func (h *ChecklistHandler) DeleteChecklist(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID checklist tidak valid"})
		return
	}

	checklist, err := h.checklistRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data checklist"})
		return
	}
	if checklist == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Checklist tidak ditemukan"})
		return
	}

	plan, err := h.actionPlanRepo.FindByID(checklist.ActionPlanID)
	if err != nil || plan == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi action plan induk"})
		return
	}

	auth := getAuthContext(c)
	if !h.authorizeActionPlanAccess(auth, plan) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola checklist action plan ini"})
		return
	}

	if err := h.checklistRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus checklist"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Checklist berhasil dihapus"})
}
