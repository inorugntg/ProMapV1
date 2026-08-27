package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"corporate-action-plan/backend/internal/middleware"
	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// ListUsers mengembalikan daftar user. Super Admin melihat semua perusahaan,
// role lain (Admin Operasional) hanya melihat user di perusahaannya sendiri.
func (h *UserHandler) ListUsers(c *gin.Context) {
	role := c.MustGet(middleware.ContextRole).(string)
	perusahaanID := c.MustGet(middleware.ContextPerusahaanID).(uint)

	filterPerusahaanID := perusahaanID
	if role == utils.RoleSuperAdmin {
		filterPerusahaanID = 0 // 0 = tidak difilter, lihat semua perusahaan
	}

	users, err := h.userRepo.List(filterPerusahaanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data user"})
		return
	}

	for i := range users {
		users[i].Password = ""
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

type createUserRequest struct {
	Nama         string `json:"nama" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=6"`
	Role         string `json:"role" binding:"required"`
	PerusahaanID uint   `json:"perusahaan_id"`
	DivisiID     *uint  `json:"divisi_id"`
	SupervisorID *uint  `json:"supervisor_id"`
}

// CreateUser menambahkan user baru. Admin Operasional/Manager hanya bisa menambahkan
// user di perusahaannya sendiri (perusahaan_id dari body diabaikan demi keamanan multi-tenant);
// Super Admin bebas menentukan perusahaan_id target.
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !utils.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	role := c.MustGet(middleware.ContextRole).(string)
	contextPerusahaanID := c.MustGet(middleware.ContextPerusahaanID).(uint)

	targetPerusahaanID := contextPerusahaanID
	if role == utils.RoleSuperAdmin {
		if req.PerusahaanID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "perusahaan_id wajib diisi"})
			return
		}
		targetPerusahaanID = req.PerusahaanID
	}

	companyExists, err := h.userRepo.PerusahaanExists(targetPerusahaanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi perusahaan"})
		return
	}
	if !companyExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Perusahaan tidak ditemukan"})
		return
	}

	if req.DivisiID != nil {
		divisiValid, err := h.userRepo.DivisiBelongsToPerusahaan(*req.DivisiID, targetPerusahaanID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi divisi"})
			return
		}
		if !divisiValid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Divisi tidak ditemukan pada perusahaan ini"})
			return
		}
	}

	exists, err := h.userRepo.EmailExists(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi email"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}

	user := models.User{
		Nama:         req.Nama,
		Email:        req.Email,
		Password:     string(hashed),
		Role:         req.Role,
		PerusahaanID: targetPerusahaanID,
		DivisiID:     req.DivisiID,
		SupervisorID: req.SupervisorID,
		Status:       "Active",
	}

	if err := h.userRepo.Create(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan user"})
		return
	}

	user.Password = ""
	c.JSON(http.StatusCreated, gin.H{"message": "User berhasil ditambahkan", "user": user})
}

type updateUserRequest struct {
	Role     *string `json:"role"`
	DivisiID *uint   `json:"divisi_id"`
	Status   *string `json:"status"`
}

// UpdateUser mengubah role/divisi/status user. Admin Operasional hanya boleh mengubah
// user di perusahaannya sendiri; Super Admin boleh mengubah user di perusahaan mana pun.
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID user tidak valid"})
		return
	}

	target, err := h.userRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data user"})
		return
	}
	if target == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	role := c.MustGet(middleware.ContextRole).(string)
	contextPerusahaanID := c.MustGet(middleware.ContextPerusahaanID).(uint)
	if role != utils.RoleSuperAdmin && target.PerusahaanID != contextPerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola user di luar perusahaan Anda"})
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role != nil {
		if !utils.IsValidRole(*req.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
			return
		}
		target.Role = *req.Role
	}
	if req.DivisiID != nil {
		divisiValid, err := h.userRepo.DivisiBelongsToPerusahaan(*req.DivisiID, target.PerusahaanID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi divisi"})
			return
		}
		if !divisiValid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Divisi tidak ditemukan pada perusahaan ini"})
			return
		}
		target.DivisiID = req.DivisiID
	}
	if req.Status != nil {
		target.Status = *req.Status
	}

	if err := h.userRepo.Update(target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui user"})
		return
	}

	target.Password = ""
	c.JSON(http.StatusOK, gin.H{"message": "User berhasil diperbarui", "user": target})
}

// DeleteUser melakukan soft delete. Admin Operasional hanya boleh menghapus user
// di perusahaannya sendiri; Super Admin boleh menghapus user di perusahaan mana pun.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID user tidak valid"})
		return
	}

	target, err := h.userRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data user"})
		return
	}
	if target == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	role := c.MustGet(middleware.ContextRole).(string)
	contextPerusahaanID := c.MustGet(middleware.ContextPerusahaanID).(uint)
	if role != utils.RoleSuperAdmin && target.PerusahaanID != contextPerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola user di luar perusahaan Anda"})
		return
	}

	if err := h.userRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User berhasil dihapus"})
}
