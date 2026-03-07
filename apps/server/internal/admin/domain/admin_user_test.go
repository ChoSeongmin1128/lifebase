package domain

import "testing"

func TestAdminUserIsSuperAdmin(t *testing.T) {
	var nilAdmin *AdminUser
	if nilAdmin.IsSuperAdmin() {
		t.Fatal("nil admin must not be super admin")
	}
	if (&AdminUser{Role: RoleAdmin}).IsSuperAdmin() {
		t.Fatal("admin role must not be super admin")
	}
	if !(&AdminUser{Role: RoleSuperAdmin}).IsSuperAdmin() {
		t.Fatal("super admin role should be super admin")
	}
}

