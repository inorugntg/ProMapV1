package service

import (
	"corporate-action-plan/backend/models"
)

type CalendarEvent struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"` // "task" | "action_plan"
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Status    string `json:"status"`
	PIC       string `json:"pic"`
}

type CalendarService struct{}

func NewCalendarService() *CalendarService {
	return &CalendarService{}
}

// GetEvents mengambil seluruh Task & Action Plan dalam lingkup akses role (company_id + role,
// pola sama dengan taskScopeQuery/actionPlanScopeQuery) sebagai daftar event kalender datar,
// memakai start_date/end_date masing-masing.
func (s *CalendarService) GetEvents(role string, perusahaanID, divisiID, userID uint) ([]CalendarEvent, error) {
	events := make([]CalendarEvent, 0)

	var tasks []models.Task
	if err := taskScopeQuery(role, perusahaanID, divisiID, userID).
		Preload("Assignee").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	for _, task := range tasks {
		pic := ""
		if task.Assignee != nil {
			pic = task.Assignee.Nama
		}
		events = append(events, CalendarEvent{
			ID:        task.ID,
			Title:     task.Judul,
			Type:      "task",
			StartDate: task.StartDate,
			EndDate:   task.EndDate,
			Status:    task.Status,
			PIC:       pic,
		})
	}

	var plans []actionPlanWithPic
	if err := actionPlanScopeQuery(role, perusahaanID, divisiID, userID).
		Select("action_plans.*, u.nama AS pic_name").
		Joins("JOIN users u ON u.id = action_plans.user_id").
		Where("action_plans.deleted_at IS NULL").
		Scan(&plans).Error; err != nil {
		return nil, err
	}
	for _, plan := range plans {
		events = append(events, CalendarEvent{
			ID:        plan.ID,
			Title:     plan.Tugas,
			Type:      "action_plan",
			StartDate: plan.StartDate,
			EndDate:   plan.EndDate,
			Status:    plan.Status,
			PIC:       plan.PicName,
		})
	}

	return events, nil
}
