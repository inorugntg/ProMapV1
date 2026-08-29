package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type ActionPlanHandler struct {
	actionPlanRepo *repository.ActionPlanRepository
	taskRepo       *repository.TaskRepository
}

func NewActionPlanHandler(actionPlanRepo *repository.ActionPlanRepository, taskRepo *repository.TaskRepository) *ActionPlanHandler {
	return &ActionPlanHandler{actionPlanRepo: actionPlanRepo, taskRepo: taskRepo}
}

// ListActionPlans mengembalikan daftar action plan sesuai lingkup akses role.
// Super Admin: semua perusahaan. Admin Operasional: seluruh divisi di perusahaannya.
// Manager: divisinya sendiri (untuk approve/reject). PIC/Staff: hanya action plan miliknya.
// Query param opsional ?task_id= untuk memfilter action plan pada satu task.
func (h *ActionPlanHandler) ListActionPlans(c *gin.Context) {
	auth := getAuthContext(c)

	filterPerusahaanID := auth.PerusahaanID
	var filterDivisiID uint
	var filterUserID uint

	switch {
	case auth.Role == utils.RoleSuperAdmin:
		filterPerusahaanID = 0
	case auth.Role == utils.RoleAdminOperasional:
		// seluruh divisi di perusahaannya
	case auth.Role == utils.RoleManager:
		filterDivisiID = auth.DivisiID
	default: // PIC / Staff
		filterUserID = auth.UserID
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

	plans, err := h.actionPlanRepo.List(filterPerusahaanID, filterDivisiID, filterUserID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data action plan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"action_plans": plans})
}

type createActionPlanRequest struct {
	TaskID     uint   `json:"task_id" binding:"required"`
	Tugas      string `json:"tugas" binding:"required"`
	OutcomeKPI string `json:"outcome_kpi"`
	Priority   string `json:"priority"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
}

// CreateActionPlan membuat action plan baru di bawah sebuah task, oleh PIC/Staff pemilik
// task tersebut (atau Manager/Admin/Super Admin dalam lingkupnya). Status awal "Pending"
// menunggu approve/reject dari Manager.
func (h *ActionPlanHandler) CreateActionPlan(c *gin.Context) {
	var req createActionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskRepo.FindByID(req.TaskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi task"})
		return
	}
	if task == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && task.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat membuat action plan di luar perusahaan Anda"})
		return
	}
	if utils.IsExecutorRole(auth.Role) && task.AssigneeID != auth.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat membuat action plan untuk task milik Anda sendiri"})
		return
	}
	if auth.Role == utils.RoleManager && (task.DivisiID == nil || *task.DivisiID != auth.DivisiID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat membuat action plan di luar divisi Anda"})
		return
	}

	plan := models.ActionPlan{
		TaskID:       task.ID,
		UserID:       auth.UserID,
		DivisiID:     task.DivisiID,
		PerusahaanID: task.PerusahaanID,
		Tugas:        req.Tugas,
		OutcomeKPI:   req.OutcomeKPI,
		Priority:     req.Priority,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		Status:       "Pending",
	}

	if err := h.actionPlanRepo.Create(&plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat action plan"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Action plan berhasil dibuat", "action_plan": plan})
}

type updateActionPlanRequest struct {
	Tugas        *string `json:"tugas"`
	OutcomeKPI   *string `json:"outcome_kpi"`
	Priority     *string `json:"priority"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	Status       *string `json:"status"`
	CatatanAdmin *string `json:"catatan_admin"`
}

// UpdateActionPlan mengedit action plan. Manager/Admin/Super Admin melakukan approve/reject
// (mengubah status menjadi "Approved"/"Rejected" beserta catatan_admin) dalam lingkupnya.
// Pemilik (PIC/Staff) hanya boleh mengubah isi rencana selagi status masih "Pending", dan
// menandai "Done" setelah action plan disetujui.
func (h *ActionPlanHandler) UpdateActionPlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID action plan tidak valid"})
		return
	}

	plan, err := h.actionPlanRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data action plan"})
		return
	}
	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Action plan tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	var req updateActionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if utils.IsExecutorRole(auth.Role) {
		if plan.UserID != auth.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat mengubah action plan milik Anda sendiri"})
			return
		}
		if req.Status != nil {
			if plan.Status != "Approved" || *req.Status != "Done" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat menandai Done setelah action plan disetujui"})
				return
			}
			plan.Status = *req.Status
		} else {
			if plan.Status != "Pending" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Action plan yang sudah diproses tidak dapat diubah isinya"})
				return
			}
			if req.Tugas != nil {
				plan.Tugas = *req.Tugas
			}
			if req.OutcomeKPI != nil {
				plan.OutcomeKPI = *req.OutcomeKPI
			}
			if req.Priority != nil {
				plan.Priority = *req.Priority
			}
			if req.StartDate != nil {
				plan.StartDate = *req.StartDate
			}
			if req.EndDate != nil {
				plan.EndDate = *req.EndDate
			}
		}
	} else {
		if auth.Role != utils.RoleSuperAdmin && plan.PerusahaanID != auth.PerusahaanID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola action plan di luar perusahaan Anda"})
			return
		}
		if auth.Role == utils.RoleManager && (plan.DivisiID == nil || *plan.DivisiID != auth.DivisiID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola action plan di luar divisi Anda"})
			return
		}

		if req.Tugas != nil {
			plan.Tugas = *req.Tugas
		}
		if req.OutcomeKPI != nil {
			plan.OutcomeKPI = *req.OutcomeKPI
		}
		if req.Priority != nil {
			plan.Priority = *req.Priority
		}
		if req.StartDate != nil {
			plan.StartDate = *req.StartDate
		}
		if req.EndDate != nil {
			plan.EndDate = *req.EndDate
		}
		if req.Status != nil {
			plan.Status = *req.Status
		}
		if req.CatatanAdmin != nil {
			plan.CatatanAdmin = *req.CatatanAdmin
		}
	}

	if err := h.actionPlanRepo.Update(plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui action plan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Action plan berhasil diperbarui", "action_plan": plan})
}

// DeleteActionPlan menghapus action plan (soft delete). Pemilik hanya boleh menghapus
// selagi status masih "Pending"; Manager/Admin/Super Admin boleh menghapus dalam lingkupnya.
func (h *ActionPlanHandler) DeleteActionPlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID action plan tidak valid"})
		return
	}

	plan, err := h.actionPlanRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data action plan"})
		return
	}
	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Action plan tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if utils.IsExecutorRole(auth.Role) {
		if plan.UserID != auth.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat menghapus action plan milik Anda sendiri"})
			return
		}
		if plan.Status != "Pending" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Action plan yang sudah diproses tidak dapat dihapus"})
			return
		}
	} else {
		if auth.Role != utils.RoleSuperAdmin && plan.PerusahaanID != auth.PerusahaanID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola action plan di luar perusahaan Anda"})
			return
		}
		if auth.Role == utils.RoleManager && (plan.DivisiID == nil || *plan.DivisiID != auth.DivisiID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola action plan di luar divisi Anda"})
			return
		}
	}

	if err := h.actionPlanRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus action plan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Action plan berhasil dihapus"})
}
