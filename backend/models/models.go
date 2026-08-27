package models

import "gorm.io/gorm"

// Perusahaan memiliki banyak Divisi, Project, dan User
type Perusahaan struct {
	gorm.Model
	Nama     string    `json:"nama" gorm:"type:varchar(191);unique;not null"`
	KodeUnik string    `json:"kode_unik" gorm:"type:varchar(191);unique;not null"`
	Divisies []Divisi  `json:"divisi"`
	Projects []Project `json:"projects"`
	Users    []User    `json:"users"`
}

// Divisi memiliki banyak User dan ActionPlan
type Divisi struct {
	gorm.Model
	PerusahaanID uint         `json:"perusahaan_id"`
	Nama         string       `json:"nama" gorm:"type:varchar(191);not null"`
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

// Project memiliki banyak ActionPlan
type Project struct {
	gorm.Model
	PerusahaanID uint         `json:"perusahaan_id"`
	UserID       uint         `json:"user_id"`
	Nama         string       `json:"nama" gorm:"type:varchar(191);not null"`
	Deskripsi    string       `json:"deskripsi" gorm:"type:varchar(500)"`
	ActionPlans  []ActionPlan `json:"action_plans" gorm:"foreignKey:ProjectID"`
}

// Action Plan (Tabel Pusat) - Menghubungkan semua entitas
type ActionPlan struct {
	gorm.Model
	ProjectID    uint   `json:"project_id"`
	UserID       uint   `json:"user_id"`
	DivisionID   uint   `json:"division_id"`   // Relasi ke Divisi
	PerusahaanID uint   `json:"perusahaan_id"` // Relasi ke Perusahaan
	Tugas        string `json:"tugas" gorm:"type:varchar(191);not null"`
	OutcomeKPI   string `json:"outcome_kpi" gorm:"type:varchar(500)"`
	Priority     string `json:"priority" gorm:"type:varchar(50)"`
	StartDate    string `json:"start_date" gorm:"type:varchar(50)"`
	EndDate      string `json:"end_date" gorm:"type:varchar(50)"`
	Status       string `json:"status" gorm:"type:varchar(50)"`
	CatatanAdmin string `json:"catatan_admin" gorm:"type:varchar(500)"`
}
