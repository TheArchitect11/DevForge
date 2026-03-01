// Package server provides the central DevForge configuration server
// with in-memory storage, REST API, and token authentication.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Organization represents a DevForge org with config and policies.
type Organization struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Config    map[string]interface{} `json:"config"`
	Policy    map[string]interface{} `json:"policy"`
	CreatedAt time.Time              `json:"createdAt"`
}

// User represents a system user.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
	OrgID string `json:"orgId"`
}

// Token represents an API token.
type Token struct {
	Value     string    `json:"token"`
	UserID    string    `json:"userId"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// AgentRecord represents a registered agent.
type AgentRecord struct {
	ID            string    `json:"id"`
	MachineID     string    `json:"machineId"`
	Hostname      string    `json:"hostname"`
	Port          int       `json:"port"`
	Version       string    `json:"version"`
	Token         string    `json:"token"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	RegisteredAt  time.Time `json:"registeredAt"`
}

// Storage defines the interface for data persistence. Currently
// backed by in-memory storage; designed for easy DB replacement.
type Storage interface {
	CreateOrg(org Organization) error
	GetOrg(id string) (*Organization, error)
	UpdateOrgConfig(id string, config map[string]interface{}) error
	UpdateOrgPolicy(id string, policy map[string]interface{}) error
	CreateToken(userID, role string) (*Token, error)
	ValidateToken(token string) (*Token, error)
	RegisterAgent(rec AgentRecord) error
	AgentHeartbeat(agentID string) error
}

// MemoryStorage is an in-memory implementation of Storage.
type MemoryStorage struct {
	orgs   map[string]Organization
	tokens map[string]Token
	agents map[string]AgentRecord
	mu     sync.RWMutex
}

// NewMemoryStorage creates an empty in-memory store.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		orgs:   make(map[string]Organization),
		tokens: make(map[string]Token),
		agents: make(map[string]AgentRecord),
	}
}

func (m *MemoryStorage) CreateOrg(org Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.orgs[org.ID]; exists {
		return fmt.Errorf("organization %q already exists", org.ID)
	}
	org.CreatedAt = time.Now()
	m.orgs[org.ID] = org
	return nil
}

func (m *MemoryStorage) GetOrg(id string) (*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	org, ok := m.orgs[id]
	if !ok {
		return nil, fmt.Errorf("organization %q not found", id)
	}
	return &org, nil
}

func (m *MemoryStorage) UpdateOrgConfig(id string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	org, ok := m.orgs[id]
	if !ok {
		return fmt.Errorf("organization %q not found", id)
	}
	org.Config = config
	m.orgs[id] = org
	return nil
}

func (m *MemoryStorage) UpdateOrgPolicy(id string, policy map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	org, ok := m.orgs[id]
	if !ok {
		return fmt.Errorf("organization %q not found", id)
	}
	org.Policy = policy
	m.orgs[id] = org
	return nil
}

func (m *MemoryStorage) CreateToken(userID, role string) (*Token, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	token := Token{
		Value:     hex.EncodeToString(tokenBytes),
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token.Value] = token
	return &token, nil
}

func (m *MemoryStorage) ValidateToken(tokenValue string) (*Token, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	token, ok := m.tokens[tokenValue]
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}
	return &token, nil
}

func (m *MemoryStorage) RegisterAgent(rec AgentRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec.RegisteredAt = time.Now()
	rec.LastHeartbeat = time.Now()
	rec.Status = "active"
	m.agents[rec.ID] = rec
	return nil
}

func (m *MemoryStorage) AgentHeartbeat(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %q not found", agentID)
	}
	agent.LastHeartbeat = time.Now()
	agent.Status = "active"
	m.agents[agentID] = agent
	return nil
}

// generateID creates a random hex ID.
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
