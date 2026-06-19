package repository

import (
	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/passwd"
)

// seedUsers creates the default accounts. Change these immediately in any real
// deployment. Default credentials: admin/admin123 and viewer/viewer123.
func seedUsers() []storeUser {
	mk := func(id, username, name string, role domain.Role, password string) storeUser {
		salt := passwd.NewSalt()
		return storeUser{
			ID:           id,
			Username:     username,
			Name:         name,
			Role:         role,
			Salt:         salt,
			PasswordHash: passwd.Hash(password, salt),
		}
	}
	return []storeUser{
		mk("usr-admin", "admin", "Administrator Finance", domain.RoleAdmin, "admin123"),
		mk("usr-viewer", "viewer", "Viewer", domain.RoleViewer, "viewer123"),
	}
}
