package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type TaskHandler struct {
	taskRepo      *repository.TaskRepository
	objectiveRepo *repository.ObjectiveRepository
	projectRepo   *repository.ProjectRepository
	userRepo      *repository.UserRepository
}

func NewTaskHandler(taskRepo *repository.TaskRepository, objectiveRepo *repository.ObjectiveRepository, projectRepo *repository.ProjectRepository, userRepo *repository.UserRepository) *TaskHandler {
	return &TaskHandler{taskRepo: taskRepo, objectiveRepo: objectiveRepo, projectRepo: projectRepo, userRepo: userRepo}
}

// ListTasks mengembalikan daftar task sesuai lingkup akses role. Super Admin melihat semua
// perusahaan, Admin Operasional melihat seluruh divisi di perusahaannya, Manager hanya
// divisinya, dan PIC/Staff hanya task yang di-assign ke dirinya (termasuk Personal Task).
// Query param opsional ?type=Task|PersonalTask untuk memfilter tipe work item.
func (h *TaskHandler) ListTasks(c *gin.Context) {
	auth := getAuthContext(c)

	filterPerusahaanID := auth.PerusahaanID
	var filterDivisiID *uint
	var filterAssigneeID uint

	switch {
	case auth.Role == utils.RoleSuperAdmin:
		filterPerusahaanID = 0
	case auth.Role == utils.RoleAdminOperasional:
		// lihat semua divisi di perusahaannya
	case auth.Role == utils.RoleManager:
		filterDivisiID = &auth.DivisiID
	default: // PIC / Staff
		filterAssigneeID = auth.UserID
	}

	tasks, err := h.taskRepo.List(filterPerusahaanID, filterDivisiID, filterAssigneeID, c.Query("type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data task"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

type createTaskRequest struct {
	ObjectiveID *uint  `json:"objective_id"`
	ProjectID   *uint  `json:"project_id"`
	AssigneeID  uint   `json:"assignee_id"`
	Type        string `json:"type"`
	Judul       string `json:"judul" binding:"required"`
	Deskripsi   string `json:"deskripsi"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

// CreateTask membuat task baru. Manager/Admin/Super Admin mendelegasikan Task resmi ke
// seorang PIC/Staff (assignee_id wajib, harus terhubung ke objective/project yang valid).
// PIC/Staff membuat Personal Task untuk dirinya sendiri -- tidak memengaruhi progres project,
// sehingga objective_id/project_id diabaikan dan assignee dipaksa ke diri sendiri.
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auth := getAuthContext(c)
	task := models.Task{
		PerusahaanID: auth.PerusahaanID,
		CreatedByID:  auth.UserID,
		Judul:        req.Judul,
		Deskripsi:    req.Deskripsi,
		Priority:     req.Priority,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
	}
	task.Status = req.Status
	if task.Status == "" {
		task.Status = "Not Started"
	}

	if utils.IsExecutorRole(auth.Role) {
		// Personal Task: milik sendiri, tidak terhubung hierarki project.
		task.Type = "PersonalTask"
		task.AssigneeID = auth.UserID
		if auth.DivisiID != 0 {
			task.DivisiID = &auth.DivisiID
		}
	} else {
		task.Type = "Task"
		if req.AssigneeID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignee_id wajib diisi untuk delegasi Task"})
			return
		}

		assignee, err := h.userRepo.FindByID(req.AssigneeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi assignee"})
			return
		}
		if assignee == nil || assignee.PerusahaanID != auth.PerusahaanID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Assignee tidak ditemukan pada perusahaan ini"})
			return
		}
		if auth.Role == utils.RoleManager && (assignee.DivisiID == nil || *assignee.DivisiID != auth.DivisiID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mendelegasikan task ke luar divisi Anda"})
			return
		}
		task.AssigneeID = assignee.ID
		task.DivisiID = assignee.DivisiID

		if req.ObjectiveID != nil {
			objective, err := h.objectiveRepo.FindByID(*req.ObjectiveID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi objective"})
				return
			}
			if objective == nil || objective.PerusahaanID != auth.PerusahaanID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Objective tidak ditemukan pada perusahaan ini"})
				return
			}
			if auth.Role == utils.RoleManager && (objective.DivisiID == nil || *objective.DivisiID != auth.DivisiID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat menggunakan objective di luar divisi Anda"})
				return
			}
			task.ObjectiveID = &objective.ID
			task.ProjectID = &objective.ProjectID
		} else if req.ProjectID != nil {
			project, err := h.projectRepo.FindByID(*req.ProjectID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi project"})
				return
			}
			if project == nil || project.PerusahaanID != auth.PerusahaanID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Project tidak ditemukan pada perusahaan ini"})
				return
			}
			if auth.Role == utils.RoleManager && (project.DivisiID == nil || *project.DivisiID != auth.DivisiID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat menggunakan project di luar divisi Anda"})
				return
			}
			task.ProjectID = &project.ID
		}
	}

	if err := h.taskRepo.Create(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat task"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Task berhasil dibuat", "task": task})
}

type updateTaskRequest struct {
	Judul      *string `json:"judul"`
	Deskripsi  *string `json:"deskripsi"`
	Status     *string `json:"status"`
	Priority   *string `json:"priority"`
	StartDate  *string `json:"start_date"`
	EndDate    *string `json:"end_date"`
	AssigneeID *uint   `json:"assignee_id"`
}

// UpdateTask mengedit task. Manager/Admin/Super Admin bisa mengedit seluruh field dalam
// lingkupnya. PIC/Staff (pemilik task) hanya boleh mengubah status tugasnya sendiri.
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID task tidak valid"})
		return
	}

	task, err := h.taskRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data task"})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if utils.IsExecutorRole(auth.Role) {
		if task.AssigneeID != auth.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat mengubah status task milik Anda sendiri"})
			return
		}
		if req.Status == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status wajib diisi"})
			return
		}
		task.Status = *req.Status
	} else {
		if auth.Role != utils.RoleSuperAdmin && task.PerusahaanID != auth.PerusahaanID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola task di luar perusahaan Anda"})
			return
		}
		if auth.Role == utils.RoleManager && (task.DivisiID == nil || *task.DivisiID != auth.DivisiID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola task di luar divisi Anda"})
			return
		}

		if req.Judul != nil {
			task.Judul = *req.Judul
		}
		if req.Deskripsi != nil {
			task.Deskripsi = *req.Deskripsi
		}
		if req.Status != nil {
			task.Status = *req.Status
		}
		if req.Priority != nil {
			task.Priority = *req.Priority
		}
		if req.StartDate != nil {
			task.StartDate = *req.StartDate
		}
		if req.EndDate != nil {
			task.EndDate = *req.EndDate
		}
		if req.AssigneeID != nil {
			assignee, err := h.userRepo.FindByID(*req.AssigneeID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi assignee"})
				return
			}
			if assignee == nil || assignee.PerusahaanID != task.PerusahaanID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Assignee tidak ditemukan pada perusahaan ini"})
				return
			}
			if auth.Role == utils.RoleManager && (assignee.DivisiID == nil || *assignee.DivisiID != auth.DivisiID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mendelegasikan task ke luar divisi Anda"})
				return
			}
			task.AssigneeID = assignee.ID
			task.DivisiID = assignee.DivisiID
		}
	}

	if err := h.taskRepo.Update(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui task"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Task berhasil diperbarui", "task": task})
}

// DeleteTask menghapus task (soft delete). Hanya Super Admin, Admin Operasional, dan Manager
// (Manager terbatas pada task di divisinya sendiri).
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID task tidak valid"})
		return
	}

	task, err := h.taskRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data task"})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && task.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola task di luar perusahaan Anda"})
		return
	}
	if auth.Role == utils.RoleManager && (task.DivisiID == nil || *task.DivisiID != auth.DivisiID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola task di luar divisi Anda"})
		return
	}

	if err := h.taskRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus task"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Task berhasil dihapus"})
}
