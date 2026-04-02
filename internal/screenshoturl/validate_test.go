package screenshoturl

import (
	"testing"
)

func TestValidateRejectsLoopbackIP(t *testing.T) {
	_, err := ValidateTargetURL("https://127.0.0.1/")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRejectsPrivateIP(t *testing.T) {
	_, err := ValidateTargetURL("http://10.0.0.1/")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRejectsMetadataIP(t *testing.T) {
	_, err := ValidateTargetURL("http://169.254.169.254/")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRejectsNonHTTP(t *testing.T) {
	_, err := ValidateTargetURL("javascript:alert(1)")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateAcceptsExampleCom(t *testing.T) {
	u, err := ValidateTargetURL("https://example.com/path")
	if err != nil {
		t.Skip("no DNS/network:", err)
	}
	if u.Hostname() != "example.com" {
		t.Fatalf("host: %q", u.Hostname())
	}
}
