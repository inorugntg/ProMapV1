package service

import (
	"math"

	"gorm.io/gorm"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type StatDistribution struct {
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type PICProgress struct {
	PICName        string  `json:"pic_name"`
	TotalTasks     int64   `json:"total_tasks"`
	CompletedTasks int64   `json:"completed_tasks"`
	CompletionRate float64 `json:"completion_rate"`
}

type DashboardStats struct {
	TotalTasks           int64                       `json:"total_tasks"`
	TotalActionPlans     int64                       `json:"total_action_plans"`
	StatusDistribution   map[string]StatDistribution `json:"status_distribution"`
	PriorityDistribution map[string]StatDistribution `json:"priority_distribution"`
	ProgressPerPIC       []PICProgress               `json:"progress_per_pic"`
	CompletionRate       float64                     `json:"completion_rate"`
}

type DashboardService struct{}

func NewDashboardService() *DashboardService {
	return &DashboardService{}
}

// taskScope membangun query Task yang sudah difilter sesuai lingkup akses role, mengikuti
// pola yang sama dengan TaskHandler.ListTasks: Super Admin melihat semua perusahaan, Admin
// Operasional melihat seluruh divisi di perusahaannya, Manager hanya divisinya, dan PIC/Staff
// hanya task miliknya sendiri.
func (s *DashboardService) taskScope(role string, perusahaanID, divisiID, userID uint) *gorm.DB {
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

// actionPlanScope sama seperti taskScope tapi untuk Action Plan (field division_id/user_id).
func (s *DashboardService) actionPlanScope(role string, perusahaanID, divisiID, userID uint) *gorm.DB {
	query := config.DB.Model(&models.ActionPlan{})

	switch {
	case role == utils.RoleSuperAdmin:
	case role == utils.RoleAdminOperasional:
		query = query.Where("perusahaan_id = ?", perusahaanID)
	case role == utils.RoleManager:
		query = query.Where("perusahaan_id = ? AND division_id = ?", perusahaanID, divisiID)
	default:
		query = query.Where("perusahaan_id = ? AND user_id = ?", perusahaanID, userID)
	}

	return query
}

func percentageOf(count, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(count)/float64(total)*10000) / 100
}

type statusCountRow struct {
	Status string
	Count  int64
}

type priorityCountRow struct {
	Priority string
	Count    int64
}

type picProgressRow struct {
	PicName   string
	Total     int64
	Completed int64
}

// GetDashboardStats mengembalikan ringkasan statistik Task & Action Plan sesuai lingkup akses
// (company_id + role) dari user yang meminta.
func (s *DashboardService) GetDashboardStats(userID uint, role string, perusahaanID uint, divisiID uint) (*DashboardStats, error) {
	var totalTasks int64
	if err := s.taskScope(role, perusahaanID, divisiID, userID).Count(&totalTasks).Error; err != nil {
		return nil, err
	}

	var totalActionPlans int64
	if err := s.actionPlanScope(role, perusahaanID, divisiID, userID).Count(&totalActionPlans).Error; err != nil {
		return nil, err
	}

	var statusRows []statusCountRow
	if err := s.taskScope(role, perusahaanID, divisiID, userID).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusRows).Error; err != nil {
		return nil, err
	}

	statusKeys := map[string]string{
		"Not Started": "not_started",
		"In Progress": "in_progress",
		"Pending":     "pending",
		"Overdue":     "overdue",
		"Complete":    "complete",
	}
	statusDistribution := map[string]StatDistribution{
		"not_started": {},
		"in_progress": {},
		"pending":     {},
		"overdue":     {},
		"complete":    {},
	}

	var completedTasks int64
	for _, row := range statusRows {
		if row.Status == "Complete" {
			completedTasks = row.Count
		}
		key, ok := statusKeys[row.Status]
		if !ok {
			continue
		}
		statusDistribution[key] = StatDistribution{Count: row.Count, Percentage: percentageOf(row.Count, totalTasks)}
	}

	var priorityRows []priorityCountRow
	if err := s.taskScope(role, perusahaanID, divisiID, userID).
		Select("priority, COUNT(*) as count").
		Group("priority").
		Scan(&priorityRows).Error; err != nil {
		return nil, err
	}

	priorityKeys := map[string]string{
		"High":   "high",
		"Medium": "medium",
		"Low":    "low",
	}
	priorityDistribution := map[string]StatDistribution{
		"high":   {},
		"medium": {},
		"low":    {},
	}

	for _, row := range priorityRows {
		key, ok := priorityKeys[row.Priority]
		if !ok {
			continue
		}
		priorityDistribution[key] = StatDistribution{Count: row.Count, Percentage: percentageOf(row.Count, totalTasks)}
	}

	progressPerPIC, err := s.getProgressPerPIC(role, perusahaanID, divisiID, userID)
	if err != nil {
		return nil, err
	}

	return &DashboardStats{
		TotalTasks:           totalTasks,
		TotalActionPlans:     totalActionPlans,
		StatusDistribution:   statusDistribution,
		PriorityDistribution: priorityDistribution,
		ProgressPerPIC:       progressPerPIC,
		CompletionRate:       percentageOf(completedTasks, totalTasks),
	}, nil
}

// getProgressPerPIC menghitung, per pemilik Action Plan (user_id), jumlah action plan total
// dan yang sudah Done, dalam lingkup akses role yang sama dengan actionPlanScope.
func (s *DashboardService) getProgressPerPIC(role string, perusahaanID, divisiID, userID uint) ([]PICProgress, error) {
	query := config.DB.Table("action_plans AS ap").
		Select("u.nama AS pic_name, COUNT(ap.id) AS total, SUM(CASE WHEN ap.status = 'Done' THEN 1 ELSE 0 END) AS completed").
		Joins("JOIN users u ON u.id = ap.user_id").
		Where("ap.deleted_at IS NULL")

	switch {
	case role == utils.RoleSuperAdmin:
	case role == utils.RoleAdminOperasional:
		query = query.Where("ap.perusahaan_id = ?", perusahaanID)
	case role == utils.RoleManager:
		query = query.Where("ap.perusahaan_id = ? AND ap.division_id = ?", perusahaanID, divisiID)
	default:
		query = query.Where("ap.perusahaan_id = ? AND ap.user_id = ?", perusahaanID, userID)
	}

	var rows []picProgressRow
	if err := query.Group("u.id, u.nama").Scan(&rows).Error; err != nil {
		return nil, err
	}

	progress := make([]PICProgress, 0, len(rows))
	for _, row := range rows {
		progress = append(progress, PICProgress{
			PICName:        row.PicName,
			TotalTasks:     row.Total,
			CompletedTasks: row.Completed,
			CompletionRate: percentageOf(row.Completed, row.Total),
		})
	}

	return progress, nil
}