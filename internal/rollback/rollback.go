// Package rollback provides a LIFO stack of compensating operations
// that are executed when a multi-step process fails partway through.
package rollback

import (
	"fmt"
	"strings"
	"sync"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/ux"
)

// Action is a single rollback step with a human-readable description.
type Action struct {
	Description string
	Fn          func() error
}

// Manager collects rollback actions and executes them in reverse order.
// It is safe for concurrent registration.
type Manager struct {
	actions []Action
	log     *logger.Logger
	mu      sync.Mutex
}

// NewManager creates a Manager.
func NewManager(log *logger.Logger) *Manager {
	return &Manager{log: log}
}

// Register adds an action to the top of the rollback stack.
// Actions are executed in reverse order (last registered → first executed).
func (m *Manager) Register(description string, fn func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.actions = append(m.actions, Action{Description: description, Fn: fn})
	m.log.Debug(fmt.Sprintf("rollback registered: %s", description))
}

// Execute runs every registered action in reverse order.
// It continues even when individual actions fail, collecting all errors
// into a combined report. The stack is cleared after execution so
// Execute is idempotent.
func (m *Manager) Execute() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.actions) == 0 {
		return nil
	}

	ux.Warning("Rolling back %d action(s) …", len(m.actions))
	m.log.Warn(fmt.Sprintf("executing %d rollback action(s)", len(m.actions)))

	var failures []string
	for i := len(m.actions) - 1; i >= 0; i-- {
		a := m.actions[i]
		ux.Step("Undoing: %s", a.Description)
		m.log.Info(fmt.Sprintf("  rollback: %s", a.Description))

		if err := a.Fn(); err != nil {
			msg := fmt.Sprintf("%s: %v", a.Description, err)
			failures = append(failures, msg)
			ux.Warning("  ↳ failed: %v", err)
			m.log.Error(fmt.Sprintf("  rollback failed: %s", msg))
		} else {
			ux.Success("  ↳ undone")
			m.log.Info(fmt.Sprintf("  rollback ok: %s", a.Description))
		}
	}

	m.actions = nil // prevent double-execution

	if len(failures) > 0 {
		return fmt.Errorf("rollback completed with %d error(s):\n  - %s",
			len(failures), strings.Join(failures, "\n  - "))
	}

	m.log.Info("all rollback actions completed")
	return nil
}

// HasActions reports whether any actions are registered.
func (m *Manager) HasActions() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.actions) > 0
}
