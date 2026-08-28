package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/service"
	"corporate-action-plan/backend/internal/utils"
)

type KanbanHandler struct {
	kanbanService  *service.KanbanService
	taskRepo       *repository.TaskRepository
	actionPlanRepo *repository.ActionPlanRepository
}

func NewKanbanHandler(kanbanService *service.KanbanService, taskRepo *repository.TaskRepository, actionPlanRepo *repository.ActionPlanRepository) *KanbanHandler {
	return &KanbanHandler{kanbanService: kanbanService, taskRepo: taskRepo, actionPlanRepo: actionPlanRepo}
}

// GetBoard mengembalikan Task & Action Plan dalam lingkup akses role, dikelompokkan per
// status ke dalam 5 kolom kanban baku (Not Started, In Progress, Pending, Overdue, Complete).
func (h *KanbanHandler) GetBoard(c *gin.Context) {
	auth := getAuthContext(c)

	board, err := h.kanbanService.GetBoard(auth.Role, auth.PerusahaanID, auth.DivisiID, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kanban"})
		return
	}

	c.JSON(http.StatusOK, board)
}

type updateKanbanStatusRequest struct {
	Type   string `json:"type" binding:"required"` // "task" | "action_plan"
	Status string `json:"status" binding:"required"`
}

// UpdateStatus mengubah status Task atau Action Plan untuk drag-and-drop di kanban board.
// Task dan Action Plan adalah tabel terpisah yang bisa memiliki ID sama, sehingga request body
// wajib menyertakan "type" untuk menentukan tabel yang dituju.
func (h *KanbanHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var req updateKanbanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auth := getAuthContext(c)

	switch req.Type {
	case "task":
		h.updateTaskStatus(c, uint(id), req.Status, auth)
	case "action_plan":
		h.updateActionPlanStatus(c, uint(id), req.Status, auth)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type harus 'task' atau 'action_plan'"})
	}
}

// updateTaskStatus menerapkan lingkup akses yang sama dengan TaskHandler.UpdateTask: PIC/Staff
// hanya boleh mengubah task miliknya sendiri, Manager terbatas pada divisinya, Admin/Super
// Admin terbatas pada perusahaannya (Super Admin bebas lintas perusahaan).
func (h *KanbanHandler) updateTaskStatus(c *gin.Context, id uint, status string, auth authContext) {
	task, err := h.taskRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data task"})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task tidak ditemukan"})
		return
	}

	if utils.IsExecutorRole(auth.Role) {
		if task.AssigneeID != auth.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat mengubah status task milik Anda sendiri"})
			return
		}
	} else {
		if auth.Role != utils.RoleSuperAdmin && task.PerusahaanID != auth.PerusahaanID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola task di luar perusahaan Anda"})
			return
		}
		if auth.Role == utils.RoleManager && (task.DivisiID == nil || *task.DivisiID != auth.DivisiID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola task di luar divisi Anda"})
			return
		}
	}

	task.Status = status
	if err := h.taskRepo.Update(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status task"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status task berhasil diperbarui", "task": task})
}

// updateActionPlanStatus menerapkan lingkup akses yang sama dengan ActionPlanHandler
// (kepemilikan/divisi/perusahaan), tapi -- berbeda dari ActionPlanHandler.UpdateActionPlan --
// sengaja tidak menerapkan gating approval (Approved -> Done) karena endpoint ini khusus untuk
// drag-and-drop bebas di kanban, bukan alur approve/reject formal.
func (h *KanbanHandler) updateActionPlanStatus(c *gin.Context, id uint, status string, auth authContext) {
	plan, err := h.actionPlanRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data action plan"})
		return
	}
	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Action plan tidak ditemukan"})
		return
	}

	if utils.IsExecutorRole(auth.Role) {
		if plan.UserID != auth.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat mengubah status action plan milik Anda sendiri"})
			return
		}
	} else {
		if auth.Role != utils.RoleSuperAdmin && plan.PerusahaanID != auth.PerusahaanID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola action plan di luar perusahaan Anda"})
			return
		}
		if auth.Role == utils.RoleManager && plan.DivisionID != auth.DivisiID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola action plan di luar divisi Anda"})
			return
		}
	}

	plan.Status = status
	if err := h.actionPlanRepo.Update(plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status action plan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status action plan berhasil diperbarui", "action_plan": plan})
}
