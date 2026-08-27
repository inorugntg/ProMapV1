package models

import "gorm.io/gorm"

// Perusahaan memiliki banyak Divisi
type Perusahaan struct {
	gorm.Model
	Nama     string    `json:"nama" gorm:"unique;not null"`
	Divisies []Divisi  `json:"divisi"`
}

// Divisi memiliki banyak User
type Divisi struct {
	gorm.Model
	PerusahaanID uint   `json:"perusahaan_id"`
	Nama         string `json:"nama" gorm:"not null"`
	Users        []User `json:"users"`
}

// User bisa jadi Bawahan atau Atasan (Self-Referencing)
type User struct {
	gorm.Model
	DivisiID     uint         `json:"divisi_id"`
	Nama         string       `json:"nama" gorm:"not null"`
	Email        string       `json:"email" gorm:"uniqueIndex;not null"`
	Password     string       `json:"-"`
	Role         string       `json:"role"` // superadmin, admin, manager, pic, magang
	SupervisorID *uint        `json:"supervisor_id"` // Pointer untuk null jika tidak punya atasan
	Projects     []Project    `json:"projects"`
	ActionPlans  []ActionPlan `json:"action_plans"`
}

// Project menampung banyak Action Plan
type Project struct {
	gorm.Model
	UserID      uint         `json:"user_id"` // Pembuat / Penanggung Jawab Project
	Nama        string       `json:"nama" gorm:"not null"`
	Deskripsi   string       `json:"deskripsi"`
	ActionPlans []ActionPlan `json:"action_plans"`
}

// Action Plan butuh Validasi Atasan
type ActionPlan struct {
	gorm.Model
	ProjectID    uint   `json:"project_id"`
	UserID       uint   `json:"user_id"` // Pelaksana (Karyawan/Magang)
	Tugas        string `json:"tugas" gorm:"not null"`
	Status       string `json:"status"`        // Pending, Approved, Rejected, Done
	CatatanAdmin string `json:"catatan_admin"` // Feedback dari atasan saat validasi
}