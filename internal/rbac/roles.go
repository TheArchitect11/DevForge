// Package rbac provides role-based access control for DevForge.
// It defines roles, permissions, and middleware for HTTP handlers.
package rbac

// Role represents a user role in the DevForge system.
type Role string

const (
	// RoleAdmin has full access to all operations.
	RoleAdmin Role = "admin"
	// RoleDeveloper can perform development operations.
	RoleDeveloper Role = "developer"
	// RoleViewer has read-only access.
	RoleViewer Role = "viewer"
)

// ValidRoles is the set of all recognized roles.
var ValidRoles = map[Role]bool{
	RoleAdmin:     true,
	RoleDeveloper: true,
	RoleViewer:    true,
}

// IsValid checks whether a role string is a recognized role.
func (r Role) IsValid() bool {
	return ValidRoles[r]
}

// String returns the string representation of the role.
func (r Role) String() string {
	return string(r)
}

// rolePermissions maps each role to the set of permissions it grants.
var rolePermissions = map[Role]map[Permission]bool{
	RoleAdmin: {
		PermInit:      true,
		PermInstall:   true,
		PermUpdate:    true,
		PermPluginRun: true,
		PermPolicy:    true,
		PermAuditRead: true,
		PermOrgManage: true,
	},
	RoleDeveloper: {
		PermInit:      true,
		PermInstall:   true,
		PermUpdate:    true,
		PermPluginRun: true,
	},
	RoleViewer: {
		PermAuditRead: true,
	},
}

// GetPermissions returns the permission set for a given role.
func GetPermissions(role Role) map[Permission]bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return map[Permission]bool{}
	}
	// Return a copy to prevent mutation.
	result := make(map[Permission]bool, len(perms))
	for k, v := range perms {
		result[k] = v
	}
	return result
}
