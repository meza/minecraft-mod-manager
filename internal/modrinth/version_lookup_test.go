package modrinth

import "testing"

func TestNewVersionHashLookup(t *testing.T) {
	lookup := NewVersionHashLookup("abc", SHA1)
	if lookup == nil {
		t.Fatalf("expected lookup")
	}
	if lookup.hash != "abc" {
		t.Fatalf("expected hash abc, got %q", lookup.hash)
	}
	if lookup.algorithm != SHA1 {
		t.Fatalf("expected algorithm sha1, got %q", lookup.algorithm)
	}
}
