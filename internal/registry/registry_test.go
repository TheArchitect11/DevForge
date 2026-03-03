package registry

import "testing"

func TestTemplate_Validate(t *testing.T) {
	tests := []struct {
		name     string
		template Template
		want     bool
	}{
		{
			name:     "valid template",
			template: Template{Name: "go-api", Description: "A Go API", URL: "https://github.com/user/repo.git"},
			want:     true,
		},
		{name: "missing name", template: Template{Description: "desc", URL: "https://example.com"}, want: false},
		{name: "missing URL", template: Template{Name: "test", Description: "desc"}, want: false},
		{name: "missing description", template: Template{Name: "test", URL: "https://example.com"}, want: false},
		{name: "all empty", template: Template{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.template.Validate(); got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegistry_ValidTemplates(t *testing.T) {
	reg := Registry{
		Templates: []Template{
			{Name: "good", Description: "valid", URL: "https://example.com/repo.git"},
			{Name: "", Description: "missing name", URL: "https://example.com"},
			{Name: "also-good", Description: "another valid", URL: "https://example.com/other.git"},
		},
	}

	valid := reg.ValidTemplates()
	if len(valid) != 2 {
		t.Errorf("ValidTemplates() returned %d, want 2", len(valid))
	}
	if valid[0].Name != "good" || valid[1].Name != "also-good" {
		t.Errorf("ValidTemplates() returned wrong templates: %+v", valid)
	}
}

func TestSearch(t *testing.T) {
	reg := &Registry{
		Templates: []Template{
			{Name: "go-api", Description: "A Go REST API", URL: "https://github.com/test/go-api.git", Tags: []string{"go", "backend"}},
			{Name: "react-app", Description: "React frontend", URL: "https://github.com/test/react.git", Tags: []string{"react", "frontend"}},
			{Name: "python-ml", Description: "Machine Learning", URL: "https://github.com/test/ml.git", Tags: []string{"python", "ai"}},
		},
	}

	client := &Client{}

	tests := []struct {
		keyword string
		count   int
	}{
		{keyword: "go", count: 1},
		{keyword: "react", count: 1},
		{keyword: "api", count: 1},
		{keyword: "frontend", count: 1},
		{keyword: "python", count: 1},
		{keyword: "nonexistent", count: 0},
		{keyword: "", count: 3},
		{keyword: "GO", count: 1}, // case insensitive
	}

	for _, tt := range tests {
		t.Run("search_"+tt.keyword, func(t *testing.T) {
			results := client.Search(reg, tt.keyword)
			if len(results) != tt.count {
				t.Errorf("Search(%q) returned %d results, want %d", tt.keyword, len(results), tt.count)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello", "hello world", false},
		{"", "", true},
		{"abc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			if got := containsIgnoreCase(tt.s, tt.substr); got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
