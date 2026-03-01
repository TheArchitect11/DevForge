package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/chinmay/devforge/internal/rbac"
)

// TokenAuthMiddleware validates Bearer tokens against the storage
// layer and injects user info into the request context.
func TokenAuthMiddleware(store Storage) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"authorization header required"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			token, err := store.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
				return
			}

			user := rbac.UserInfo{
				ID:   token.UserID,
				Name: token.UserID,
				Role: rbac.Role(token.Role),
			}

			ctx := rbac.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// contextKey for storage in context.
type ctxKey string

const storageKey ctxKey = "storage"

// WithStorage adds storage to the context.
func WithStorage(ctx context.Context, store Storage) context.Context {
	return context.WithValue(ctx, storageKey, store)
}

// StorageFromContext retrieves storage from the context.
func StorageFromContext(ctx context.Context) Storage {
	store, _ := ctx.Value(storageKey).(Storage)
	return store
}
