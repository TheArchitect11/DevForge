package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/policy"
	"github.com/chinmay/devforge/internal/rbac"
	"github.com/chinmay/devforge/internal/remote"
	"github.com/chinmay/devforge/internal/server"
)

// ── Policy Blocking Tests ──────────────────────────────────
func TestPolicyBlocksDependency(t *testing.T) {
	log, err := logger.New(false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Close()

	p := &policy.Policy{
		AllowedDependencies: []string{"node", "git"},
	}
	engine := policy.NewEngine(p, log)

	// Allowed dependency.
	if v := engine.CheckDependency("node", "18"); v != nil {
		t.Errorf("expected node to be allowed, got violation: %s", v.Message)
	}

	// Blocked dependency.
	if v := engine.CheckDependency("redis", "7"); v == nil {
		t.Error("expected redis to be blocked by policy")
	}
}

func TestPolicyBlocksTemplate(t *testing.T) {
	log, err := logger.New(false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Close()

	p := &policy.Policy{
		BlockedTemplates: []string{"unknown-repo"},
	}
	engine := policy.NewEngine(p, log)

	if v := engine.CheckTemplate("https://github.com/safe-org/template"); v != nil {
		t.Errorf("expected safe template to pass, got: %s", v.Message)
	}

	if v := engine.CheckTemplate("https://github.com/unknown-repo/template"); v == nil {
		t.Error("expected unknown-repo to be blocked")
	}
}

func TestPolicyMaxNodeVersion(t *testing.T) {
	log, err := logger.New(false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Close()

	p := &policy.Policy{
		MaxNodeVersion: 20,
	}
	engine := policy.NewEngine(p, log)

	if v := engine.CheckDependency("node", "18"); v != nil {
		t.Errorf("expected node 18 to pass, got: %s", v.Message)
	}

	if v := engine.CheckDependency("node", "22"); v == nil {
		t.Error("expected node 22 to be blocked (max=20)")
	}
}

// ── RBAC Enforcement Tests ─────────────────────────────────
func TestRBACPermissions(t *testing.T) {
	tests := []struct {
		role rbac.Role
		perm rbac.Permission
		want bool
	}{
		{rbac.RoleAdmin, rbac.PermInit, true},
		{rbac.RoleAdmin, rbac.PermPluginRun, true},
		{rbac.RoleDeveloper, rbac.PermInit, true},
		{rbac.RoleDeveloper, rbac.PermOrgManage, false},
		{rbac.RoleViewer, rbac.PermInit, false},
		{rbac.RoleViewer, rbac.PermAuditRead, true},
	}

	for _, tc := range tests {
		got := rbac.HasPermission(tc.role, tc.perm)
		if got != tc.want {
			t.Errorf("HasPermission(%s, %s) = %v, want %v", tc.role, tc.perm, got, tc.want)
		}
	}
}

func TestRBACMiddleware(t *testing.T) {
	handler := rbac.RequirePermission(rbac.PermInit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	// No user in context → 401.
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without user, got %d", rec.Code)
	}

	// Viewer (no init permission) → 403.
	req = httptest.NewRequest("GET", "/test", nil)
	ctx := rbac.WithUser(req.Context(), rbac.UserInfo{ID: "viewer1", Role: rbac.RoleViewer})
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer, got %d", rec.Code)
	}

	// Admin → 200.
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = rbac.WithUser(req.Context(), rbac.UserInfo{ID: "admin1", Role: rbac.RoleAdmin})
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d", rec.Code)
	}
}

// ── Remote Execution Flow Tests ─────────────────────────────
func TestRemoteProtocol(t *testing.T) {
	req := remote.Request{
		Command:     "init",
		ProjectName: "test-project",
		Version:     "1.0.0",
		DryRun:      true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var decoded remote.Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if decoded.Command != "init" || decoded.ProjectName != "test-project" {
		t.Error("round-trip failed for remote request")
	}
}

// ── Server Handler Tests ────────────────────────────────────
func TestServerCreateOrg(t *testing.T) {
	store := server.NewMemoryStorage()

	r := chi.NewRouter()
	server.RegisterHandlers(r, store)

	// Create a token first.
	body := `{"userId":"user1","role":"admin"}`
	req := httptest.NewRequest("POST", "/auth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create token returned %d", rec.Code)
	}

	var tokenResp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&tokenResp)
	token := tokenResp["token"].(string)

	// Create org.
	body = `{"name":"TestOrg"}`
	req = httptest.NewRequest("POST", "/org", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create org returned %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerAgentRegistration(t *testing.T) {
	store := server.NewMemoryStorage()

	r := chi.NewRouter()
	server.RegisterHandlers(r, store)

	body := `{"machineId":"host-1","hostname":"test-host","port":8443,"version":"1.0.0"}`
	req := httptest.NewRequest("POST", "/api/v1/agents/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("agent register returned %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["agentId"] == "" || resp["token"] == "" {
		t.Error("expected agentId and token in registration response")
	}
}
