package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/internal/handler"
	"corporate-action-plan/backend/internal/middleware"
	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

func main() {
	// 1. Load environment variables dari file .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// 2. Koneksi ke database MySQL
	config.ConnectDatabase()

	// 3. Auto-Migrate: Buat tabel otomatis berdasarkan struct models
	err = config.DB.AutoMigrate(
		&models.Perusahaan{},
		&models.Divisi{},
		&models.User{},
		&models.Project{},
		&models.ActionPlan{},
	)
	if err != nil {
		log.Fatal("Gagal AutoMigrate: ", err)
	}
	log.Println("✅ Tabel berhasil dibuat/dimigrasi!")

	// 4. Inisialisasi Repository & Handler
	userRepo := repository.NewUserRepository()
	authHandler := handler.NewAuthHandler(userRepo)
	userHandler := handler.NewUserHandler(userRepo)

	// 5. Inisialisasi Router Gin
	r := gin.Default()

	// CORS: izinkan frontend React (Vite dev server) memanggil API ini dari origin berbeda
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 6. Contoh route sederhana untuk cek server
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "Server ProMaP Backend Berjalan!",
			"database": "Terhubung",
		})
	})

	// 7. Route API
	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.GET("/profile", middleware.AuthRequired(), authHandler.Profile)
		}

		users := api.Group("/users")
		users.Use(middleware.AuthRequired())
		{
			users.GET("", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional), userHandler.ListUsers)
			users.POST("", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional, utils.RoleManager), userHandler.CreateUser)
			users.PUT("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional), userHandler.UpdateUser)
			users.DELETE("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional), userHandler.DeleteUser)
		}
	}

	// 8. Jalankan server di port 8080
	r.Run(":8080")
}