package init

import (
	"testing"
)

func TestComponent(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	c := Component()

	if c.Name != "cmd.init.short" {
		t.Errorf("unexpected component name: %s", c.Name)
	}

	if c.Model == nil {
		t.Error("component model must not be nil")
	}
}
