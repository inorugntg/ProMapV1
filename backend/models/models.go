package models

import "gorm.io/gorm"

// Perusahaan memiliki banyak Divisi, Project, dan User
type Perusahaan struct {
	gorm.Model
	Nama               string    `json:"nama" gorm:"type:varchar(191);unique;not null"`
	KodeUnik           string    `json:"kode_unik" gorm:"type:varchar(191);unique;not null"`
	StatusSubscription string    `json:"status_subscription" gorm:"type:varchar(20);not null;default:'basic'"`
	Divisies           []Divisi  `json:"divisi"`
	Projects           []Project `json:"projects"`
	Users              []User    `json:"users"`
}

// Divisi memiliki banyak User dan ActionPlan
type Divisi struct {
	gorm.Model
	PerusahaanID uint         `json:"perusahaan_id"`
	Nama         string       `json:"nama" gorm:"type:varchar(191);not null"`
	Deskripsi    string       `json:"deskripsi" gorm:"type:varchar(500)"`
	Users        []User       `json:"users"`
	ActionPlans  []ActionPlan `json:"action_plans" gorm:"foreignKey:DivisionID"`
}

// User bisa jadi Bawahan atau Atasan (Self-Referencing)
type User struct {
	gorm.Model
	DivisiID     *uint        `json:"divisi_id"`
	PerusahaanID uint         `json:"perusahaan_id"`
	Nama         string       `json:"nama" gorm:"type:varchar(191);not null"`
	Email        string       `json:"email" gorm:"type:varchar(191);uniqueIndex;not null"`
	Password     string       `json:"-" gorm:"type:varchar(191)"`
	Role         string       `json:"role" gorm:"type:varchar(50)"`
	Status       string       `json:"status" gorm:"type:varchar(20);not null;default:'Active'"`
	SupervisorID *uint        `json:"supervisor_id"`
	Projects     []Project    `json:"projects"`
	ActionPlans  []ActionPlan `json:"action_plans"`
}

// Project adalah tujuan besar (Work Item level 1). Berisi banyak Objective.
type Project struct {
	gorm.Model
	PerusahaanID uint        `json:"perusahaan_id"`
	DivisiID     *uint       `json:"divisi_id"`
	UserID       uint        `json:"user_id"` // pembuat/pemilik project (Manager/Admin)
	Nama         string      `json:"nama" gorm:"type:varchar(191);not null"`
	Deskripsi    string      `json:"deskripsi" gorm:"type:varchar(500)"`
	Status       string      `json:"status" gorm:"type:varchar(50);not null;default:'Not Started'"`
	StartDate    string      `json:"start_date" gorm:"type:varchar(50)"`
	EndDate      string      `json:"end_date" gorm:"type:varchar(50)"`
	Objectives   []Objective `json:"objectives" gorm:"foreignKey:ProjectID"`
}

// Objective adalah turunan tujuan besar Project (Work Item level 2). Berisi banyak Task.
type Objective struct {
	gorm.Model
	ProjectID    uint   `json:"project_id" gorm:"not null"`
	PerusahaanID uint   `json:"perusahaan_id"`
	DivisiID     *uint  `json:"divisi_id"`
	Judul        string `json:"judul" gorm:"type:varchar(191);not null"`
	Deskripsi    string `json:"deskripsi" gorm:"type:varchar(500)"`
	Status       string `json:"status" gorm:"type:varchar(50);not null;default:'Not Started'"`
	StartDate    string `json:"start_date" gorm:"type:varchar(50)"`
	EndDate      string `json:"end_date" gorm:"type:varchar(50)"`
	Tasks        []Task `json:"tasks" gorm:"foreignKey:ObjectiveID"`
}

// Task adalah delegasi resmi dari Manager/Admin ke seorang PIC/Staff (Work Item level 3),
// atau -- jika Type == "PersonalTask" -- pekerjaan pribadi PIC yang tidak memengaruhi
// progres Project/Objective (ObjectiveID & ProjectID boleh kosong untuk kasus ini).
type Task struct {
	gorm.Model
	ObjectiveID  *uint        `json:"objective_id"`
	ProjectID    *uint        `json:"project_id"`
	PerusahaanID uint         `json:"perusahaan_id"`
	DivisiID     *uint        `json:"divisi_id"`
	AssigneeID   uint         `json:"assignee_id" gorm:"not null"`                          // PIC/Staff pemilik tugas
	CreatedByID  uint         `json:"created_by_id"`                                        // Manager/Admin pemberi delegasi (diri sendiri jika PersonalTask)
	Type         string       `json:"type" gorm:"type:varchar(20);not null;default:'Task'"` // "Task" | "PersonalTask"
	Judul        string       `json:"judul" gorm:"type:varchar(191);not null"`
	Deskripsi    string       `json:"deskripsi" gorm:"type:varchar(500)"`
	Status       string       `json:"status" gorm:"type:varchar(50);not null;default:'Not Started'"`
	Priority     string       `json:"priority" gorm:"type:varchar(50)"`
	StartDate    string       `json:"start_date" gorm:"type:varchar(50)"`
	EndDate      string       `json:"end_date" gorm:"type:varchar(50)"`
	Assignee     *User        `json:"assignee,omitempty" gorm:"foreignKey:AssigneeID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	CreatedBy    *User        `json:"created_by,omitempty" gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	ActionPlans  []ActionPlan `json:"action_plans" gorm:"foreignKey:TaskID"`
}

// ActionPlan adalah langkah taktis yang dibuat oleh pemilik Task (Work Item level 4).
type ActionPlan struct {
	gorm.Model
	TaskID       uint        `json:"task_id" gorm:"not null"`
	UserID       uint        `json:"user_id"` // PIC/Staff pembuat action plan
	DivisionID   uint        `json:"division_id"`
	PerusahaanID uint        `json:"perusahaan_id"`
	Tugas        string      `json:"tugas" gorm:"type:varchar(191);not null"`
	OutcomeKPI   string      `json:"outcome_kpi" gorm:"type:varchar(500)"`
	Priority     string      `json:"priority" gorm:"type:varchar(50)"`
	StartDate    string      `json:"start_date" gorm:"type:varchar(50)"`
	EndDate      string      `json:"end_date" gorm:"type:varchar(50)"`
	Status       string      `json:"status" gorm:"type:varchar(50);not null;default:'Pending'"` // Pending, Approved, Rejected, Done
	CatatanAdmin string      `json:"catatan_admin" gorm:"type:varchar(500)"`
	Checklists   []Checklist `json:"checklists" gorm:"foreignKey:ActionPlanID"`
}

// Checklist adalah rincian terkecil dalam Action Plan (Work Item level 5).
type Checklist struct {
	gorm.Model
	ActionPlanID uint   `json:"action_plan_id" gorm:"not null"`
	TaskID       uint   `json:"task_id"` // denormalisasi dari ActionPlan.TaskID untuk mempercepat filter
	PerusahaanID uint   `json:"perusahaan_id"`
	NamaItem     string `json:"nama_item" gorm:"type:varchar(191);not null"`
	Status       string `json:"status" gorm:"type:varchar(20);not null;default:'Not Done'"` // "Done" | "Not Done"
}

// Evidence adalah bukti penyelesaian yang diunggah PIC/Staff untuk sebuah Action Plan.
type Evidence struct {
	gorm.Model
	ActionPlanID uint   `json:"action_plan_id" gorm:"not null"`
	TaskID       uint   `json:"task_id"` // denormalisasi dari ActionPlan.TaskID
	PerusahaanID uint   `json:"perusahaan_id"`
	UserID       uint   `json:"user_id"` // User yang mengupload evidence
	Judul        string `json:"judul" gorm:"type:varchar(191);not null"`
	Deskripsi    string `json:"deskripsi" gorm:"type:varchar(500)"`
	Link         string `json:"link" gorm:"type:varchar(500)"`                             // Link Google Drive / file
	Status       string `json:"status" gorm:"type:varchar(20);not null;default:'Pending'"` // Pending, Approved, Rejected
	CatatanAdmin string `json:"catatan_admin" gorm:"type:varchar(500)"`
}

// Proposal adalah ide dari Staff yang menunggu persetujuan Manager.
type Proposal struct {
	gorm.Model
	PerusahaanID    uint   `json:"perusahaan_id"`
	DivisiID        *uint  `json:"divisi_id"`
	ProjectID       *uint  `json:"project_id"`                    // opsional, proposal boleh berdiri sendiri sebagai ide baru
	CreatedByID     uint   `json:"created_by_id" gorm:"not null"` // Staff pengaju
	Judul           string `json:"judul" gorm:"type:varchar(191);not null"`
	Deskripsi       string `json:"deskripsi" gorm:"type:varchar(500)"`
	Status          string `json:"status" gorm:"type:varchar(30);not null;default:'Pending Approval'"` // Pending Approval, Approved, Rejected
	ApprovedByID    *uint  `json:"approved_by_id"`
	CatatanApproval string `json:"catatan_approval" gorm:"type:varchar(500)"`
	CreatedBy       *User  `json:"created_by,omitempty" gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	ApprovedBy      *User  `json:"approved_by,omitempty" gorm:"foreignKey:ApprovedByID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
