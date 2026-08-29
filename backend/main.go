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
	"corporate-action-plan/backend/internal/service"
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
		&models.Evidence{},
		&models.Proposal{},
		&models.Notification{},
		&models.Tamu{},
		&models.ActivityLog{},
	)
	if err != nil {
		log.Fatal("Gagal AutoMigrate: ", err)
	}
	log.Println("✅ Tabel berhasil dibuat/dimigrasi!")

	// 4. Inisialisasi Repository & Handler
	userRepo := repository.NewUserRepository()
	companyRepo := repository.NewCompanyRepository()
	divisionRepo := repository.NewDivisionRepository()
	projectRepo := repository.NewProjectRepository()
	objectiveRepo := repository.NewObjectiveRepository()
	taskRepo := repository.NewTaskRepository()
	actionPlanRepo := repository.NewActionPlanRepository()
	checklistRepo := repository.NewChecklistRepository()
	evidenceRepo := repository.NewEvidenceRepository()
	proposalRepo := repository.NewProposalRepository()

	authHandler := handler.NewAuthHandler(userRepo)
	userHandler := handler.NewUserHandler(userRepo)
	companyHandler := handler.NewCompanyHandler(companyRepo)
	divisionHandler := handler.NewDivisionHandler(divisionRepo, companyRepo)
	projectHandler := handler.NewProjectHandler(projectRepo, userRepo)
	objectiveHandler := handler.NewObjectiveHandler(objectiveRepo, projectRepo, userRepo)
	taskHandler := handler.NewTaskHandler(taskRepo, objectiveRepo, projectRepo, userRepo)
	actionPlanHandler := handler.NewActionPlanHandler(actionPlanRepo, taskRepo)
	checklistHandler := handler.NewChecklistHandler(checklistRepo, actionPlanRepo)
	evidenceHandler := handler.NewEvidenceHandler(evidenceRepo, actionPlanRepo)
	proposalHandler := handler.NewProposalHandler(proposalRepo, projectRepo)

	notificationService := service.NewNotificationService()
	overdueService := service.NewOverdueService(notificationService)
	dashboardService := service.NewDashboardService()
	kanbanService := service.NewKanbanService()
	calendarService := service.NewCalendarService()

	overdueHandler := handler.NewOverdueHandler(overdueService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	kanbanHandler := handler.NewKanbanHandler(kanbanService, taskRepo, actionPlanRepo)
	calendarHandler := handler.NewCalendarHandler(calendarService)

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

		// Perusahaan (Company)
		companies := api.Group("/companies")
		companies.Use(middleware.AuthRequired())
		{
			companies.GET("", companyHandler.ListCompanies)
			companies.POST("", middleware.RequireRole(utils.RoleSuperAdmin), companyHandler.CreateCompany)
			companies.PUT("/:id", middleware.RequireRole(utils.RoleSuperAdmin), companyHandler.UpdateCompany)
			companies.DELETE("/:id", middleware.RequireRole(utils.RoleSuperAdmin), companyHandler.DeleteCompany)
		}

		// Divisi (Division)
		divisions := api.Group("/divisions")
		divisions.Use(middleware.AuthRequired())
		{
			divisions.GET("", divisionHandler.ListDivisions)
			divisions.POST("", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional), divisionHandler.CreateDivision)
			divisions.PUT("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional), divisionHandler.UpdateDivision)
			divisions.DELETE("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional), divisionHandler.DeleteDivision)
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

		// Evidence (bukti penyelesaian yang diunggah PIC/Staff untuk sebuah Action Plan)
		evidences := api.Group("/evidences")
		evidences.Use(middleware.AuthRequired())
		{
			evidences.GET("", evidenceHandler.ListEvidences)
			evidences.POST("", evidenceHandler.CreateEvidence)
			evidences.PUT("/:id", evidenceHandler.UpdateEvidence)
			evidences.DELETE("/:id", evidenceHandler.DeleteEvidence)
		}

		// Proposal (ide dari Staff, menunggu approve/reject Manager)
		proposals := api.Group("/proposals")
		proposals.Use(middleware.AuthRequired())
		{
			proposals.GET("", proposalHandler.ListProposals)
			proposals.POST("", middleware.RequireRole(utils.RoleStaff), proposalHandler.CreateProposal)
			proposals.PUT("/:id", middleware.RequireRole(utils.RoleSuperAdmin, utils.RoleAdminOperasional, utils.RoleManager), proposalHandler.UpdateProposal)
		}

		// Overdue (pengecekan status overdue Task & Action Plan)
		overdue := api.Group("/overdue")
		overdue.Use(middleware.AuthRequired())
		{
			overdue.POST("/check", middleware.RequireRole(utils.RoleSuperAdmin), overdueHandler.CheckOverdue)
		}

		// Notification (notifikasi in-app milik user yang login)
		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthRequired())
		{
			notifications.GET("", notificationHandler.ListNotifications)
			notifications.POST("/read-all", notificationHandler.MarkAllRead)
			notifications.PUT("/:id/read", notificationHandler.MarkRead)
		}

		// Dashboard (statistik ringkasan Task & Action Plan sesuai lingkup akses role)
		dashboard := api.Group("/dashboard")
		dashboard.Use(middleware.AuthRequired())
		{
			dashboard.GET("", dashboardHandler.GetDashboard)
		}

		// Kanban (papan Task & Action Plan per status, untuk tampilan drag-and-drop)
		kanban := api.Group("/kanban")
		kanban.Use(middleware.AuthRequired())
		{
			kanban.GET("", kanbanHandler.GetBoard)
			kanban.PUT("/:id/status", kanbanHandler.UpdateStatus)
		}

		// Calendar (Task & Action Plan berdasarkan start_date/end_date)
		calendar := api.Group("/calendar")
		calendar.Use(middleware.AuthRequired())
		{
			calendar.GET("", calendarHandler.GetEvents)
		}
	}

	// 8. Background worker: jalankan pengecekan overdue otomatis setiap hari jam 00:01
	go runOverdueWorker(overdueService)

	// 9. Jalankan server di port 8080
	r.Run(":8080")
}

// runOverdueWorker menjalankan CheckAll() setiap hari tepat jam 00:01 waktu lokal server.
// Dijalankan sebagai goroutine terpisah agar tidak memblokir server HTTP.
func runOverdueWorker(overdueService *service.OverdueService) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 1, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		time.Sleep(time.Until(next))

		overdueTasks, overdueActionPlans, err := overdueService.CheckAll()
		if err != nil {
			log.Println("Gagal menjalankan pengecekan overdue terjadwal:", err)
			continue
		}
		log.Printf("Pengecekan overdue terjadwal selesai: %d task, %d action plan diubah menjadi Overdue\n", overdueTasks, overdueActionPlans)
	}
}
