package service

import (
	"corporate-action-plan/backend/models"
)

type KanbanItem struct {
	ID       uint   `json:"id"`
	Type     string `json:"type"` // "task" | "action_plan" -- dibutuhkan agar PUT /api/kanban/:id/status tahu tabel mana yang dituju
	Title    string `json:"title"`
	PIC      string `json:"pic"`
	Priority string `json:"priority"`
}

type KanbanColumn struct {
	Status string       `json:"status"`
	Items  []KanbanItem `json:"items"`
}

type KanbanBoard struct {
	Columns []KanbanColumn `json:"columns"`
}

// kanbanStatuses adalah 5 kolom baku papan kanban, sesuai status Task.
var kanbanStatuses = []string{"Not Started", "In Progress", "Pending", "Overdue", "Complete"}

type KanbanService struct{}

func NewKanbanService() *KanbanService {
	return &KanbanService{}
}

type actionPlanWithPic struct {
	models.ActionPlan `gorm:"embedded"`
	PicName           string
}

// GetBoard mengambil seluruh Task & Action Plan dalam lingkup akses role (company_id + role,
// pola sama dengan taskScopeQuery/actionPlanScopeQuery), dikelompokkan per status ke dalam
// 5 kolom kanban baku. Task dipetakan langsung dari field Status-nya. Action Plan (status asli:
// Pending/Approved/Rejected/Done/Overdue) dipetakan: "Done" -> "Complete", "Pending"/"Overdue"
// dipakai apa adanya karena sudah cocok dengan kolom baku; "Approved"/"Rejected" adalah status
// approval-workflow (bukan progres pengerjaan) sehingga tidak relevan ditampilkan di kanban.
func (s *KanbanService) GetBoard(role string, perusahaanID, divisiID, userID uint) (*KanbanBoard, error) {
	grouped := make(map[string][]KanbanItem, len(kanbanStatuses))
	for _, status := range kanbanStatuses {
		grouped[status] = []KanbanItem{}
	}

	var tasks []models.Task
	if err := taskScopeQuery(role, perusahaanID, divisiID, userID).
		Preload("Assignee").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	for _, task := range tasks {
		if _, ok := grouped[task.Status]; !ok {
			continue
		}
		pic := ""
		if task.Assignee != nil {
			pic = task.Assignee.Nama
		}
		grouped[task.Status] = append(grouped[task.Status], KanbanItem{
			ID:       task.ID,
			Type:     "task",
			Title:    task.Judul,
			PIC:      pic,
			Priority: task.Priority,
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
		status := plan.Status
		if status == "Done" {
			status = "Complete"
		}
		if _, ok := grouped[status]; !ok {
			continue
		}
		grouped[status] = append(grouped[status], KanbanItem{
			ID:       plan.ID,
			Type:     "action_plan",
			Title:    plan.Tugas,
			PIC:      plan.PicName,
			Priority: plan.Priority,
		})
	}

	columns := make([]KanbanColumn, 0, len(kanbanStatuses))
	for _, status := range kanbanStatuses {
		columns = append(columns, KanbanColumn{Status: status, Items: grouped[status]})
	}

	return &KanbanBoard{Columns: columns}, nil
}
