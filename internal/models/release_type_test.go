package models

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestReleaseTypeMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		release  ReleaseType
		expected string
	}{
		{"Alpha", Alpha, "alpha"},
		{"Beta", Beta, "beta"},
		{"release", Release, "release"},
		{"Invalid", ReleaseType("invalid"), "invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := json.Marshal(test.release)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(actual) != fmt.Sprintf(`"%s"`, test.expected) {
				t.Fatalf("expected %s, got %s", test.expected, string(actual))
			}
		})
	}
}

func TestAllReleaseTypes(t *testing.T) {
	expected := []ReleaseType{Alpha, Beta, Release}
	actual := AllReleaseTypes()
	if len(expected) != len(actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
	for i, e := range expected {
		if e != actual[i] {
			t.Fatalf("expected %v, got %v", e, actual[i])
		}
	}
}
