package httpapi

import "testing"

func TestAliasRegex(t *testing.T) {
	cases := []struct {
		alias string
		ok    bool
	}{
		{"abcd", true},
		{"a_b-C1", true},
		{"a", false},
		{"abc", false},
		{"this-alias-is-way-too-long-for-the-allowed-range", false},
		{"has space", false},
		{"has.dot", false},
	}
	for _, c := range cases {
		if got := aliasRe.MatchString(c.alias); got != c.ok {
			t.Fatalf("alias %q ok=%v, got %v", c.alias, c.ok, got)
		}
	}
}

func TestIsValidURL(t *testing.T) {
	cases := []struct {
		raw string
		ok  bool
	}{
		{"https://example.com", true},
		{"http://example.com/path?q=1", true},
		{"ftp://example.com", false},
		{"example.com", false},
		{"https://", false},
		{"https://exa mple.com", false},
	}
	for _, c := range cases {
		if got := isValidURL(c.raw); got != c.ok {
			t.Fatalf("url %q ok=%v, got %v", c.raw, c.ok, got)
		}
	}
}

