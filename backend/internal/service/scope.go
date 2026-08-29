package service

import (
	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

// taskScopeQuery membangun query Task yang sudah difilter sesuai lingkup akses role, mengikuti
// pola yang sama dengan TaskHandler.ListTasks: Super Admin melihat semua perusahaan, Admin
// Operasional melihat seluruh divisi di perusahaannya, Manager hanya divisinya, dan PIC/Staff
// (dan role lain di luar itu) hanya task miliknya sendiri.
func taskScopeQuery(role string, perusahaanID, divisiID, userID uint) *gorm.DB {
	query := config.DB.Model(&models.Task{})

	switch {
	case role == utils.RoleSuperAdmin:
		// tidak difilter, lihat semua perusahaan
	case role == utils.RoleAdminOperasional:
		query = query.Where("perusahaan_id = ?", perusahaanID)
	case role == utils.RoleManager:
		query = query.Where("perusahaan_id = ? AND divisi_id = ?", perusahaanID, divisiID)
	default: // PIC / Staff / Magang
		query = query.Where("perusahaan_id = ? AND assignee_id = ?", perusahaanID, userID)
	}

	return query
}

// actionPlanScopeQuery sama seperti taskScopeQuery tapi untuk Action Plan (field
// divisi_id/user_id, bukan divisi_id/assignee_id seperti Task).
func actionPlanScopeQuery(role string, perusahaanID, divisiID, userID uint) *gorm.DB {
	query := config.DB.Model(&models.ActionPlan{})

	switch {
	case role == utils.RoleSuperAdmin:
	case role == utils.RoleAdminOperasional:
		query = query.Where("perusahaan_id = ?", perusahaanID)
	case role == utils.RoleManager:
		query = query.Where("perusahaan_id = ? AND divisi_id = ?", perusahaanID, divisiID)
	default:
		query = query.Where("perusahaan_id = ? AND user_id = ?", perusahaanID, userID)
	}

	return query
}
