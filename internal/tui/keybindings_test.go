package tui

import "testing"

func TestToggleKeyBinding(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	binding := Toggle()
	keys := binding.Keys()
	if len(keys) != 1 || keys[0] != " " {
		t.Fatalf("expected toggle keys to include space, got %v", keys)
	}

	help := binding.Help()
	if help.Key != "key.space" {
		t.Fatalf("expected toggle help key to use key.space, got %q", help.Key)
	}
	if help.Desc != "key.help.toggle" {
		t.Fatalf("expected toggle help description to use key.help.toggle, got %q", help.Desc)
	}
}
