package utils

// Daftar role yang dikenal sistem (RBAC)
const (
	RoleSuperAdmin       = "Super Admin"
	RoleAdminOperasional = "Admin Operasional"
	RoleManager          = "Manager"
	RolePIC              = "PIC"
	RoleStaff            = "Staff"
	RoleMagang           = "Magang"
)

// AllRoles berisi seluruh role valid untuk keperluan validasi input
var AllRoles = []string{RoleSuperAdmin, RoleAdminOperasional, RoleManager, RolePIC, RoleStaff, RoleMagang}

// IsManagementRole mengecek apakah role tergolong level manajemen
// (berhak membuat/mengedit Project, Objective, Task, dan approve Proposal/ActionPlan).
func IsManagementRole(role string) bool {
	return role == RoleSuperAdmin || role == RoleAdminOperasional || role == RoleManager
}

// IsExecutorRole mengecek apakah role adalah pelaksana tugas (PIC/Staff) yang
// aksesnya dibatasi hanya pada pekerjaan miliknya sendiri.
func IsExecutorRole(role string) bool {
	return role == RolePIC || role == RoleStaff
}

// IsValidRole mengecek apakah sebuah string role dikenal oleh sistem
func IsValidRole(role string) bool {
	for _, r := range AllRoles {
		if r == role {
			return true
		}
	}
	return false
}
