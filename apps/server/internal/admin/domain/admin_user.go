package domain

import "time"

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

type AdminUser struct {
	ID        string
	UserID    string
	Role      Role
	IsActive  bool
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (a *AdminUser) IsSuperAdmin() bool {
	return a != nil && a.Role == RoleSuperAdmin
}
