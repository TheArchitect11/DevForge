package semver

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Version
		wantErr bool
	}{
		{name: "major only", input: "18", want: Version{Major: 18, Raw: "18"}},
		{name: "major.minor", input: "18.2", want: Version{Major: 18, Minor: 2, Raw: "18.2"}},
		{name: "full semver", input: "1.2.3", want: Version{Major: 1, Minor: 2, Patch: 3, Raw: "1.2.3"}},
		{name: "v prefix", input: "v1.2.3", want: Version{Major: 1, Minor: 2, Patch: 3, Raw: "v1.2.3"}},
		{name: "prerelease", input: "1.0.0-beta.1", want: Version{Major: 1, PreRelease: "beta.1", Raw: "1.0.0-beta.1"}},
		{name: "latest keyword", input: "latest", want: Version{Raw: "latest"}},
		{name: "empty string", input: "", want: Version{Raw: ""}},
		{name: "invalid input", input: "not-a-version", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor ||
				got.Patch != tt.want.Patch || got.PreRelease != tt.want.PreRelease {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "equal", a: "1.2.3", b: "1.2.3", want: 0},
		{name: "major greater", a: "2.0.0", b: "1.0.0", want: 1},
		{name: "major less", a: "1.0.0", b: "2.0.0", want: -1},
		{name: "minor greater", a: "1.3.0", b: "1.2.0", want: 1},
		{name: "patch greater", a: "1.2.4", b: "1.2.3", want: 1},
		{name: "prerelease less than release", a: "1.0.0-rc.1", b: "1.0.0", want: -1},
		{name: "release greater than prerelease", a: "1.0.0", b: "1.0.0-rc.1", want: 1},
		{name: "both prerelease alpha sort", a: "1.0.0-alpha", b: "1.0.0-beta", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			va, _ := Parse(tt.a)
			vb, _ := Parse(tt.b)
			got := va.Compare(vb)
			if got != tt.want {
				t.Errorf("(%s).Compare(%s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVersion_MajorMatches(t *testing.T) {
	v18, _ := Parse("18")
	v18_2, _ := Parse("18.2.1")
	v20, _ := Parse("20.0.0")

	if !v18.MajorMatches(v18_2) {
		t.Error("expected 18 to match 18.2.1")
	}
	if v18.MajorMatches(v20) {
		t.Error("expected 18 to NOT match 20.0.0")
	}
}

func TestVersion_IsZero(t *testing.T) {
	zero := Version{}
	if !zero.IsZero() {
		t.Error("empty Version should be zero")
	}
	v, _ := Parse("1.0.0")
	if v.IsZero() {
		t.Error("parsed version should not be zero")
	}
}

func TestVersion_String(t *testing.T) {
	v, _ := Parse("1.2.3")
	if s := v.String(); s != "1.2.3" {
		t.Errorf("String() = %q, want %q", s, "1.2.3")
	}
	vPre, _ := Parse("1.0.0-beta.1")
	if s := vPre.String(); s != "1.0.0-beta.1" {
		t.Errorf("String() = %q, want %q", s, "1.0.0-beta.1")
	}
}
