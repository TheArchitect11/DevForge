package installer

import "testing"

func TestIsLatest(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"", true},
		{"latest", true},
		{"20", false},
		{"1.2.3", false},
		{"v20.0.0", false},
		{"LATEST", false}, // case-sensitive — only lowercase "latest" is treated as latest
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := isLatest(tt.version)
			if got != tt.want {
				t.Errorf("isLatest(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
