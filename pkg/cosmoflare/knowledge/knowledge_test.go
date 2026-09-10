package knowledge

import "testing"

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		path string
		want bool
	}{
		{"exact", "/zones/{zone_id}/rulesets", "/zones/z1/rulesets", true},
		{"param mismatch length", "/zones/{zone_id}/rulesets", "/zones/z1/rulesets/extra", false},
		{"two params", "/zones/{z}/rulesets/{r}/rules", "/zones/z1/rulesets/r1/rules", true},
		{"literal segment differs", "/zones/{z}/dns", "/accounts/a1/dns", false},
		{"scope star consumes rest", "/zones/{zone_id}/rulesets*", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint", true},
		{"star requires prefix", "/zones/{zone_id}/rulesets*", "/zones/z1/other", false},
		{"empty param segment", "/zones/{z}/rulesets", "/zones//rulesets", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchPath(tt.tmpl, tt.path); got != tt.want {
				t.Fatalf("matchPath(%q, %q) = %v, want %v", tt.tmpl, tt.path, got, tt.want)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/client/v4/zones/z1/rulesets", "/zones/z1/rulesets"},
		{"/zones/z1/rulesets", "/zones/z1/rulesets"},
		{"/client/v4/client/v4/zones", "/client/v4/zones"}, // strip one prefix only
	}
	for _, tt := range tests {
		if got := NormalizePath(tt.in); got != tt.want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
