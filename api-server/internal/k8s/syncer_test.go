package k8s

import (
	"testing"
)

func TestMapPhaseToDBStatus(t *testing.T) {
	tests := []struct {
		phase    string
		expected string
	}{
		{"Pending", "pending"},
		{"Analyzing", "analyzing"},
		{"Scheduling", "scheduling"},
		{"Running", "running"},
		{"Succeeded", "completed"},
		{"Failed", "failed"},
		{"Cancelled", "cancelled"},
		{"Unknown", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		got := MapPhaseToDBStatus(tt.phase)
		if got != tt.expected {
			t.Errorf("MapPhaseToDBStatus(%q) = %q, want %q", tt.phase, got, tt.expected)
		}
	}
}
