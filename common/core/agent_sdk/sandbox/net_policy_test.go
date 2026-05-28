package sandbox

import "testing"

func TestDomainAllowListMatchesWildcardPrefixes(t *testing.T) {
	policy := NewDomainAllowList("10.*", "192.168.*", "*.example.com")

	for _, host := range []string{"10.1.2.3", "192.168.0.10", "api.example.com"} {
		if err := policy.Validate(host); err != nil {
			t.Fatalf("Validate(%q) = %v, want allowed", host, err)
		}
	}
	if err := policy.Validate("172.20.0.1"); err == nil {
		t.Fatal("Validate(172.20.0.1) succeeded, want denied")
	}
}
