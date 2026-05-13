package rollback

import (
	"errors"
	"fmt"
	"testing"

	"github.com/chinmay/devforge/internal/logger"
)

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(false)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func TestExecute_LIFO_Order(t *testing.T) {
	log := newTestLogger(t)
	m := NewManager(log)

	order := []int{}
	for i := 1; i <= 4; i++ {
		i := i
		m.Register(fmt.Sprintf("action %d", i), func() error {
			order = append(order, i)
			return nil
		})
	}

	if err := m.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	// Expect reverse registration order: 4, 3, 2, 1
	want := []int{4, 3, 2, 1}
	for j, got := range order {
		if got != want[j] {
			t.Errorf("position %d: got action %d, want %d", j, got, want[j])
		}
	}
}

func TestExecute_Empty(t *testing.T) {
	m := NewManager(newTestLogger(t))
	if err := m.Execute(); err != nil {
		t.Errorf("Execute() on empty manager should return nil, got %v", err)
	}
}

func TestExecute_ContinuesAfterFailure(t *testing.T) {
	m := NewManager(newTestLogger(t))

	ran := make([]bool, 3)
	m.Register("ok-first", func() error { ran[0] = true; return nil })
	m.Register("failing", func() error { ran[1] = true; return errors.New("boom") })
	m.Register("ok-last", func() error { ran[2] = true; return nil })

	err := m.Execute()
	// Should return a combined error but still run all actions.
	if err == nil {
		t.Error("expected error from failing action, got nil")
	}
	for i, r := range ran {
		if !r {
			t.Errorf("action[%d] was not executed despite prior failure", i)
		}
	}
}

func TestExecute_Idempotent(t *testing.T) {
	m := NewManager(newTestLogger(t))
	calls := 0
	m.Register("once", func() error { calls++; return nil })

	m.Execute()
	m.Execute() // second call should be a no-op

	if calls != 1 {
		t.Errorf("Execute() should only run actions once; ran %d times", calls)
	}
}

func TestHasActions(t *testing.T) {
	m := NewManager(newTestLogger(t))
	if m.HasActions() {
		t.Error("HasActions() should be false on empty manager")
	}
	m.Register("x", func() error { return nil })
	if !m.HasActions() {
		t.Error("HasActions() should be true after Register()")
	}
	m.Execute()
	if m.HasActions() {
		t.Error("HasActions() should be false after Execute()")
	}
}
