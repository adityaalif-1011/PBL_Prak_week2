package helper

import "testing"

func TestPermissionSet_Can(t *testing.T) {
	raw := map[string][]string{
		"admin": {"student:list", "student:delete"},
		"staff": {"student:list"},
		"user":  {},
	}
	perms := NewPermissionSet(raw)

	// admin punya student:list
	if !perms.Can("admin", "student:list") {
		t.Error("admin harus punya student:list")
	}

	// staff punya student:list
	if !perms.Can("staff", "student:list") {
		t.Error("staff harus punya student:list")
	}

	// staff TIDAK punya student:delete
	if perms.Can("staff", "student:delete") {
		t.Error("staff tidak boleh punya student:delete")
	}

	// role tidak dikenal → tolak
	if perms.Can("hacker", "student:list") {
		t.Error("role tidak dikenal harus tolak")
	}

	// permission tidak dikenal → tolak
	if perms.Can("admin", "student:hack") {
		t.Error("permission tidak dikenal harus tolak")
	}

	// nil PermissionSet → tolak
	var nilPerms *PermissionSet
	if nilPerms.Can("admin", "student:list") {
		t.Error("nil PermissionSet harus tolak")
	}
}
