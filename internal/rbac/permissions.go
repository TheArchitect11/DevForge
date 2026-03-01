package rbac

import "fmt"

// Permission represents a specific action that can be authorized.
type Permission string

const (
	// PermInit allows project initialization.
	PermInit Permission = "init"
	// PermInstall allows dependency installation.
	PermInstall Permission = "install"
	// PermUpdate allows auto-update operations.
	PermUpdate Permission = "update"
	// PermPluginRun allows plugin execution.
	PermPluginRun Permission = "plugin-run"
	// PermPolicy allows policy management.
	PermPolicy Permission = "policy"
	// PermAuditRead allows reading audit logs.
	PermAuditRead Permission = "audit-read"
	// PermOrgManage allows organization management.
	PermOrgManage Permission = "org-manage"
)

// HasPermission checks whether the given role is authorized for the
// specified permission.
func HasPermission(role Role, perm Permission) bool {
	perms := GetPermissions(role)
	return perms[perm]
}

// Authorize checks whether the given role has the required permission.
// Returns an error with a clear message if access is denied.
func Authorize(role Role, perm Permission) error {
	if !role.IsValid() {
		return fmt.Errorf("access denied: unknown role %q", role)
	}
	if !HasPermission(role, perm) {
		return fmt.Errorf("access denied: role %q does not have permission %q", role, perm)
	}
	return nil
}
