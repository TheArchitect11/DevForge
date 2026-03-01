package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterHandlers mounts all REST API routes on the given router.
func RegisterHandlers(r chi.Router, store Storage) {
	// Public endpoint: create tokens.
	r.Post("/auth/token", handleCreateToken(store))

	// Agent registration (no auth required for initial registration).
	r.Post("/api/v1/agents/register", handleAgentRegister(store))
	r.Post("/api/v1/agents/heartbeat", handleAgentHeartbeat(store))

	// Authenticated routes.
	r.Group(func(r chi.Router) {
		r.Use(TokenAuthMiddleware(store))

		r.Post("/org", handleCreateOrg(store))
		r.Get("/org/{id}/config", handleGetOrgConfig(store))
		r.Post("/org/{id}/policy", handleSetOrgPolicy(store))
	})
}

// ── POST /auth/token ────────────────────────────────────────

type tokenRequest struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

func handleCreateToken(store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.UserID == "" || req.Role == "" {
			jsonError(w, "userId and role are required", http.StatusBadRequest)
			return
		}

		token, err := store.CreateToken(req.UserID, req.Role)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, token)
	}
}

// ── POST /org ───────────────────────────────────────────────

type createOrgRequest struct {
	Name string `json:"name"`
}

func handleCreateOrg(store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createOrgRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			jsonError(w, "name is required", http.StatusBadRequest)
			return
		}

		org := Organization{
			ID:     generateID(),
			Name:   req.Name,
			Config: make(map[string]interface{}),
			Policy: make(map[string]interface{}),
		}

		if err := store.CreateOrg(org); err != nil {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}

		jsonResponse(w, http.StatusCreated, org)
	}
}

// ── GET /org/{id}/config ────────────────────────────────────

func handleGetOrgConfig(store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		org, err := store.GetOrg(id)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"orgId":  org.ID,
			"name":   org.Name,
			"config": org.Config,
		})
	}
}

// ── POST /org/{id}/policy ───────────────────────────────────

func handleSetOrgPolicy(store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var policy map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := store.UpdateOrgPolicy(id, policy); err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"status": "policy updated",
			"orgId":  id,
		})
	}
}

// ── Agent Registration ──────────────────────────────────────

type agentRegisterRequest struct {
	MachineID string `json:"machineId"`
	Hostname  string `json:"hostname"`
	Port      int    `json:"port"`
	Version   string `json:"version"`
}

func handleAgentRegister(store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req agentRegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		agentID := generateID()
		tokenBytes := make([]byte, 32)
		rand.Read(tokenBytes)
		token := hex.EncodeToString(tokenBytes)

		rec := AgentRecord{
			ID:        agentID,
			MachineID: req.MachineID,
			Hostname:  req.Hostname,
			Port:      req.Port,
			Version:   req.Version,
			Token:     token,
		}

		if err := store.RegisterAgent(rec); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"agentId": agentID,
			"token":   token,
		})
	}
}

// ── Agent Heartbeat ─────────────────────────────────────────

type heartbeatRequest struct {
	AgentID   string `json:"agentId"`
	MachineID string `json:"machineId"`
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
}

func handleAgentHeartbeat(store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req heartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := store.AgentHeartbeat(req.AgentID); err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// ── Helpers ─────────────────────────────────────────────────

func jsonResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// generateHandlerID generates a unique handler-scoped ID — kept package-level
// to avoid conflict with storage.go generateID via different scope.
func init() {
	// Package-level init ensures generateID from storage.go is available.
	_ = fmt.Sprintf // ensure fmt is used
}
