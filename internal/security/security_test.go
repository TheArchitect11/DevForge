package security

import "testing"

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid simple", input: "node", wantErr: false},
		{name: "valid with dash", input: "my-project", wantErr: false},
		{name: "valid with underscore", input: "my_project", wantErr: false},
		{name: "valid with dot", input: "file.txt", wantErr: false},
		{name: "valid with at", input: "node@20", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "directory traversal", input: "../evil", wantErr: true},
		{name: "starts with dot", input: ".hidden", wantErr: true},
		{name: "shell injection", input: "name; rm -rf", wantErr: true},
		{name: "spaces", input: "my project", wantErr: true},
		{name: "too long", input: string(make([]byte, 256)), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateName(%q) expected error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateName(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid https", input: "https://github.com/user/repo.git", wantErr: false},
		{name: "valid http", input: "http://example.com/repo", wantErr: false},
		{name: "ftp scheme", input: "ftp://evil.com/hax", wantErr: true},
		{name: "file scheme", input: "file:///etc/passwd", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "no host", input: "https://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateURL(%q) expected error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateURL(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "normal", input: "hello", want: "hello"},
		{name: "trim spaces", input: "  hello  ", want: "hello"},
		{name: "null bytes", input: "he\x00llo", want: "hello"},
		{name: "combined", input: "  he\x00llo  ", want: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeInput(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeInput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
