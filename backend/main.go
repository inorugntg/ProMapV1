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
		&models.Objective{},
		&models.Task{},
		&models.ActionPlan{},
		&models.Checklist{},
		&models.Proposal{},
	)
	if err != nil {
		log.Fatal("Gagal AutoMigrate: ", err)
	}
	log.Println("✅ Tabel berhasil dibuat/dimigrasi!")

	// 4. Inisialisasi Repository & Handler
	userRepo := repository.NewUserRepository()
	projectRepo := repository.NewProjectRepository()
	objectiveRepo := repository.NewObjectiveRepository()
	taskRepo := repository.NewTaskRepository()
	actionPlanRepo := repository.NewActionPlanRepository()
	checklistRepo := repository.NewChecklistRepository()
	proposalRepo := repository.NewProposalRepository()

	authHandler := handler.NewAuthHandler(userRepo)
	userHandler := handler.NewUserHandler(userRepo)
	projectHandler := handler.NewProjectHandler(projectRepo, userRepo)
	objectiveHandler := handler.NewObjectiveHandler(objectiveRepo, projectRepo, userRepo)
	taskHandler := handler.NewTaskHandler(taskRepo, objectiveRepo, projectRepo, userRepo)
	actionPlanHandler := handler.NewActionPlanHandler(actionPlanRepo, taskRepo)
	checklistHandler := handler.NewChecklistHandler(checklistRepo, actionPlanRepo)
	proposalHandler := handler.NewProposalHandler(proposalRepo, projectRepo)

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

		// Project (Work Item level 1)
		projects := api.Group("/projects")
		projects.Use(middleware.AuthRequired())
		{
			projects.GET("", projectHandler.ListProjects)
			projects.POST("", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional, utils.RoleManager), projectHandler.CreateProject)
			projects.PUT("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional, utils.RoleManager), projectHandler.UpdateProject)
			projects.DELETE("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional), projectHandler.DeleteProject)
		}

		// Objective (Work Item level 2, turunan Project -- dibutuhkan agar hierarki bisa dibentuk)
		objectives := api.Group("/objectives")
		objectives.Use(middleware.AuthRequired())
		{
			objectives.GET("", objectiveHandler.ListObjectives)
			objectives.POST("", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional, utils.RoleManager), objectiveHandler.CreateObjective)
			objectives.PUT("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional, utils.RoleManager), objectiveHandler.UpdateObjective)
			objectives.DELETE("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional), objectiveHandler.DeleteObjective)
		}

		// Task (Work Item level 3: delegasi resmi Manager/Admin, atau Personal Task milik PIC/Staff)
		tasks := api.Group("/tasks")
		tasks.Use(middleware.AuthRequired())
		{
			tasks.GET("", taskHandler.ListTasks)
			tasks.POST("", taskHandler.CreateTask)
			tasks.PUT("/:id", taskHandler.UpdateTask)
			tasks.DELETE("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional, utils.RoleManager), taskHandler.DeleteTask)
		}

		// Action Plan (Work Item level 4, dibuat oleh pemilik Task)
		actionPlans := api.Group("/action-plans")
		actionPlans.Use(middleware.AuthRequired())
		{
			actionPlans.GET("", actionPlanHandler.ListActionPlans)
			actionPlans.POST("", actionPlanHandler.CreateActionPlan)
			actionPlans.PUT("/:id", actionPlanHandler.UpdateActionPlan)
			actionPlans.DELETE("/:id", actionPlanHandler.DeleteActionPlan)
		}

		// Checklist (Work Item level 5, rincian terkecil dalam Action Plan)
		checklists := api.Group("/checklists")
		checklists.Use(middleware.AuthRequired())
		{
			checklists.GET("", checklistHandler.ListChecklists)
			checklists.POST("", checklistHandler.CreateChecklist)
			checklists.PUT("/:id", checklistHandler.UpdateChecklist)
			checklists.DELETE("/:id", checklistHandler.DeleteChecklist)
		}

		// Proposal (ide dari Staff, menunggu approve/reject Manager)
		proposals := api.Group("/proposals")
		proposals.Use(middleware.AuthRequired())
		{
			proposals.GET("", proposalHandler.ListProposals)
			proposals.POST("", middleware.RequireRole(utils.RoleStaff), proposalHandler.CreateProposal)
			proposals.PUT("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional, utils.RoleManager), proposalHandler.UpdateProposal)
		}
	}

	// 8. Jalankan server di port 8080
	r.Run(":8080")
}
