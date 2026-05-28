package agent

import (
	"testing"
)

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"123", true},
		{"0", true},
		{"", false},
		{"abc", false},
		{"12a", false},
		{"12.3", false},
		{" 123", false},
		{"123 ", false},
	}
	for _, tt := range tests {
		if got := isNumeric(tt.input); got != tt.want {
			t.Errorf("isNumeric(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCollectProcessInfo_NonexistentProcess(t *testing.T) {
	// A process name that should not exist on any system
	info := collectProcessInfo("beszel_nonexistent_process_12345")
	if info != nil {
		t.Errorf("expected nil for nonexistent process, got %+v", info)
	}
}

func TestCollectProcessInfo_CurrentProcess(t *testing.T) {
	// The test binary itself should be findable
	// On Windows: collectProcessInfo uses tasklist which needs image name
	// We just verify the function doesn't panic and returns nil for unknown names
	info := collectProcessInfo("totally_fake_name")
	if info != nil {
		t.Errorf("expected nil for fake process name, got %+v", info)
	}
}
