package modrinth

import "testing"

func TestNewVersionHashLookup(t *testing.T) {
	lookup := NewVersionHashLookup("abc", Sha1)
	if lookup == nil {
		t.Fatalf("expected lookup")
	}
	if lookup.hash != "abc" {
		t.Fatalf("expected hash abc, got %q", lookup.hash)
	}
	if lookup.algorithm != Sha1 {
		t.Fatalf("expected algorithm sha1, got %q", lookup.algorithm)
	}
}
