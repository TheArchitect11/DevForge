package rbac

import (
	"context"
	"net/http"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// userContextKey stores the authenticated user in request context.
	userContextKey contextKey = "rbac_user"
	// roleContextKey stores the authenticated user's role.
	roleContextKey contextKey = "rbac_role"
)

// UserInfo represents an authenticated user in the request context.
type UserInfo struct {
	ID   string
	Name string
	Role Role
}

// WithUser returns a new context containing the user info.
func WithUser(ctx context.Context, user UserInfo) context.Context {
	ctx = context.WithValue(ctx, userContextKey, user)
	ctx = context.WithValue(ctx, roleContextKey, user.Role)
	return ctx
}

// UserFromContext extracts user info from the context.
func UserFromContext(ctx context.Context) (UserInfo, bool) {
	user, ok := ctx.Value(userContextKey).(UserInfo)
	return user, ok
}

// RequirePermission returns HTTP middleware that enforces a specific
// permission. If the user in context does not have the required
// permission, the request is rejected with 403 Forbidden.
func RequirePermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			if err := Authorize(user.Role, perm); err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
