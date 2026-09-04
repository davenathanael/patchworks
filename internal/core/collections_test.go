package core

import (
	"testing"

	"github.com/carlmjohnson/be"
)

func TestCollectionRoleAllows(t *testing.T) {
	perms := []Permission{PermView, PermManageBookmarks, PermEditCollection, PermDeleteCollection, PermManageMembers}
	tests := []struct {
		name    string
		role    CollectionRole
		allowed map[Permission]bool
	}{
		{"owner allows every permission", RoleOwner, map[Permission]bool{
			PermView: true, PermManageBookmarks: true, PermEditCollection: true,
			PermDeleteCollection: true, PermManageMembers: true,
		}},
		{"editor can view, manage bookmarks, and edit details", RoleEditor, map[Permission]bool{
			PermView: true, PermManageBookmarks: true, PermEditCollection: true,
		}},
		{"viewer can only view", RoleViewer, map[Permission]bool{
			PermView: true,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, p := range perms {
				be.Equal(t, tt.allowed[p], tt.role.Allows(p))
			}
		})
	}
}

func TestUnknownCollectionRoleAllowsNothing(t *testing.T) {
	perms := []Permission{PermView, PermManageBookmarks, PermEditCollection, PermDeleteCollection, PermManageMembers}
	for _, role := range []CollectionRole{"", "admin", "OWNER"} {
		for _, p := range perms {
			be.False(t, role.Allows(p))
		}
	}
}
