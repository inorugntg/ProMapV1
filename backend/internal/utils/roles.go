package utils

// Daftar role yang dikenal sistem (RBAC)
const (
	RoleSuperAdmin       = "Super Admin"
	RoleAdminOperasional = "Admin Operasional"
	RoleManager          = "Manager"
	RolePIC              = "PIC"
	RoleMagang           = "Magang"
)

// AllRoles berisi seluruh role valid untuk keperluan validasi input
var AllRoles = []string{RoleSuperAdmin, RoleAdminOperasional, RoleManager, RolePIC, RoleMagang}

// IsValidRole mengecek apakah sebuah string role dikenal oleh sistem
func IsValidRole(role string) bool {
	for _, r := range AllRoles {
		if r == role {
			return true
		}
	}
	return false
}
