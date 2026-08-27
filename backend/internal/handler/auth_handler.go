package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"corporate-action-plan/backend/internal/middleware"
	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
}

func NewAuthHandler(userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

type registerRequest struct {
	Nama                 string `json:"nama" binding:"required"`
	Email                string `json:"email" binding:"required,email"`
	Password             string `json:"password" binding:"required,min=6"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required"`
	Role                 string `json:"role" binding:"required"`
	PerusahaanID         uint   `json:"perusahaan_id" binding:"required"`
	DivisiID             *uint  `json:"divisi_id"`
}

// Register mendaftarkan user baru. Untuk semua role (termasuk Super Admin), perusahaan_id
// wajib merujuk ke perusahaan yang sudah ada -- Super Admin bebas memilih perusahaan mana pun,
// sedangkan role lain secara konvensi didaftarkan oleh admin perusahaan yang bersangkutan.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Password != req.PasswordConfirmation {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password dan konfirmasi password tidak sama"})
		return
	}

	if !utils.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	companyExists, err := h.userRepo.PerusahaanExists(req.PerusahaanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi perusahaan"})
		return
	}
	if !companyExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Perusahaan tidak ditemukan"})
		return
	}

	if req.DivisiID != nil {
		divisiValid, err := h.userRepo.DivisiBelongsToPerusahaan(*req.DivisiID, req.PerusahaanID)
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
		PerusahaanID: req.PerusahaanID,
		DivisiID:     req.DivisiID,
		Status:       "Active",
	}

	if err := h.userRepo.Create(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan user"})
		return
	}

	user.Password = ""
	c.JSON(http.StatusCreated, gin.H{"message": "Registrasi berhasil", "user": user})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login memvalidasi kredensial dan mengembalikan JWT + data user (tanpa password).
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses login"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	if user.Status != "Active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun Anda tidak aktif, silakan hubungi administrator"})
		return
	}

	var divisiID uint
	if user.DivisiID != nil {
		divisiID = *user.DivisiID
	}
	token, err := utils.GenerateToken(user.ID, user.Role, user.PerusahaanID, divisiID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
		return
	}

	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// Profile mengembalikan data user yang sedang login berdasarkan JWT yang dikirim.
func (h *AuthHandler) Profile(c *gin.Context) {
	userID := c.MustGet(middleware.ContextUserID).(uint)

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data profile"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"user": user})
}
