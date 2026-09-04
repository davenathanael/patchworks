package core

// CollectionRole is a user's role within a collection.
type CollectionRole string

const (
	RoleOwner  CollectionRole = "owner"
	RoleEditor CollectionRole = "editor"
	RoleViewer CollectionRole = "viewer"
)

// Permission is an action on a collection that a role may or may not allow.
type Permission string

const (
	PermView             Permission = "view"
	PermManageBookmarks  Permission = "manage_bookmarks"
	PermEditCollection   Permission = "edit_collection"
	PermDeleteCollection Permission = "delete_collection"
	PermManageMembers    Permission = "manage_members"
)

// Allows reports whether the role grants the permission.
func (r CollectionRole) Allows(p Permission) bool {
	switch r {
	case RoleOwner:
		return true
	case RoleEditor:
		return p == PermView || p == PermManageBookmarks || p == PermEditCollection
	case RoleViewer:
		return p == PermView
	}
	return false
}
