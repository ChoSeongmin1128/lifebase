package bootstrap

import (
	"context"
	"testing"
	"time"
)

func TestTodoBootstrapper(t *testing.T) {
	b := NewTodoBootstrapper(nil)
	if b == nil {
		t.Fatal("expected bootstrapper instance")
	}
	if err := b.BootstrapUser(context.Background(), "user-1", time.Now()); err != nil {
		t.Fatalf("bootstrap user should be noop: %v", err)
	}
}

