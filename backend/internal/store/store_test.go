package store

import (
	"testing"

	"github.com/trello-clone/backend/internal/models"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"My Workspace", "my-workspace"},
		{"  Hello   World ", "hello-world"},
		{"!!!", "workspace"},
		{"", "workspace"},
	}
	for _, tc := range tests {
		if got := Slugify(tc.in); got != tc.want {
			t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCanManage(t *testing.T) {
	if !CanManage(models.RoleOwner) || !CanManage(models.RoleAdmin) {
		t.Fatal("owner and admin should manage")
	}
	if CanManage(models.RoleMember) {
		t.Fatal("member should not manage")
	}
}
