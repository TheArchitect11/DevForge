package rollback

import (
	"errors"
	"testing"

	"github.com/chinmay/devforge/internal/logger"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	log, err := logger.New(false, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	t.Cleanup(func() { log.Close() })
	return NewManager(log)
}

func TestManager_EmptyExecute(t *testing.T) {
	m := newTestManager(t)
	if m.HasActions() {
		t.Error("new manager should have no actions")
	}
	if err := m.Execute(); err != nil {
		t.Errorf("Execute() on empty manager should return nil, got: %v", err)
	}
}

func TestManager_LIFO_Order(t *testing.T) {
	m := newTestManager(t)
	var order []int

	m.Register("first", func() error { order = append(order, 1); return nil })
	m.Register("second", func() error { order = append(order, 2); return nil })
	m.Register("third", func() error { order = append(order, 3); return nil })

	if !m.HasActions() {
		t.Error("manager should have actions after registration")
	}

	if err := m.Execute(); err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	// LIFO: third → second → first
	expected := []int{3, 2, 1}
	if len(order) != len(expected) {
		t.Fatalf("executed %d actions, want %d", len(order), len(expected))
	}
	for i, v := range order {
		if v != expected[i] {
			t.Errorf("action[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestManager_PartialFailure(t *testing.T) {
	m := newTestManager(t)

	m.Register("succeeds", func() error { return nil })
	m.Register("fails", func() error { return errors.New("rollback error") })
	m.Register("also succeeds", func() error { return nil })

	err := m.Execute()
	if err == nil {
		t.Error("Execute() should return error when an action fails")
	}
}

func TestManager_ClearsAfterExecute(t *testing.T) {
	m := newTestManager(t)
	m.Register("action", func() error { return nil })
	_ = m.Execute()

	if m.HasActions() {
		t.Error("actions should be cleared after Execute()")
	}

	// Second execute should be a no-op
	if err := m.Execute(); err != nil {
		t.Errorf("second Execute() should return nil, got: %v", err)
	}
}
