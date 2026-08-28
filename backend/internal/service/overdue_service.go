package service

import (
	"time"

	"corporate-action-plan/backend/config"
	"corporate-action-plan/backend/models"
)

// dateLayout adalah format yang dipakai pada kolom StartDate/EndDate (varchar) di
// Task/ActionPlan, yaitu ISO 8601 (YYYY-MM-DD) sehingga bisa dibandingkan langsung
// sebagai string secara leksikografis.
const dateLayout = "2006-01-02"

type OverdueService struct {
	notificationService *NotificationService
}

func NewOverdueService(notificationService *NotificationService) *OverdueService {
	return &OverdueService{notificationService: notificationService}
}

// CheckOverdueTasks mengubah status seluruh Task yang end_date-nya sudah lewat hari ini
// dan belum Complete/Overdue menjadi "Overdue", lalu mengirim notifikasi ke assignee-nya.
// Mengembalikan jumlah task yang diubah.
func (s *OverdueService) CheckOverdueTasks() (int, error) {
	today := time.Now().Format(dateLayout)

	var tasks []models.Task
	err := config.DB.
		Where("end_date <> ''").
		Where("end_date < ?", today).
		Where("status <> ?", "Complete").
		Where("status <> ?", "Overdue").
		Find(&tasks).Error
	if err != nil {
		return 0, err
	}

	changed := 0
	for _, task := range tasks {
		task.Status = "Overdue"
		if err := config.DB.Save(&task).Error; err != nil {
			continue
		}
		changed++

		if s.notificationService != nil && task.AssigneeID != 0 {
			_ = s.notificationService.Create(
				task.AssigneeID,
				"Task Terlambat",
				"Task \""+task.Judul+"\" telah melewati batas waktu ("+task.EndDate+") dan berstatus Overdue.",
				"overdue_task",
			)
		}
	}

	return changed, nil
}

// CheckOverdueActionPlans mengubah status seluruh Action Plan yang end_date-nya sudah lewat
// hari ini dan belum Done/Overdue menjadi "Overdue", lalu mengirim notifikasi ke pemiliknya.
// Mengembalikan jumlah action plan yang diubah.
func (s *OverdueService) CheckOverdueActionPlans() (int, error) {
	today := time.Now().Format(dateLayout)

	var actionPlans []models.ActionPlan
	err := config.DB.
		Where("end_date <> ''").
		Where("end_date < ?", today).
		Where("status <> ?", "Done").
		Where("status <> ?", "Overdue").
		Find(&actionPlans).Error
	if err != nil {
		return 0, err
	}

	changed := 0
	for _, plan := range actionPlans {
		plan.Status = "Overdue"
		if err := config.DB.Save(&plan).Error; err != nil {
			continue
		}
		changed++

		if s.notificationService != nil && plan.UserID != 0 {
			_ = s.notificationService.Create(
				plan.UserID,
				"Action Plan Terlambat",
				"Action plan \""+plan.Tugas+"\" telah melewati batas waktu ("+plan.EndDate+") dan berstatus Overdue.",
				"overdue_action_plan",
			)
		}
	}

	return changed, nil
}

// CheckAll menjalankan pengecekan overdue untuk Task dan Action Plan sekaligus.
func (s *OverdueService) CheckAll() (overdueTasks int, overdueActionPlans int, err error) {
	overdueTasks, err = s.CheckOverdueTasks()
	if err != nil {
		return 0, 0, err
	}

	overdueActionPlans, err = s.CheckOverdueActionPlans()
	if err != nil {
		return overdueTasks, 0, err
	}

	return overdueTasks, overdueActionPlans, nil
}
