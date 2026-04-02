package screenshoturl

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

const maxURLLen = 2048

// ValidateTargetURL checks that raw is a safe http(s) URL for server-side fetching
// (screenshot). It resolves the host and rejects loopback, private, and link-local IPs.
func ValidateTargetURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("missing url")
	}
	if len(raw) > maxURLLen {
		return nil, errors.New("url too long")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("only http and https are allowed")
	}
	if u.Host == "" {
		return nil, errors.New("missing host")
	}
	if u.User != nil {
		return nil, errors.New("credentials in url are not allowed")
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return nil, errors.New("missing host")
	}
	if host == "localhost" {
		return nil, errors.New("host not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return nil, errors.New("address not allowed")
		}
		return u, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("cannot resolve host")
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return nil, errors.New("host resolves to a non-public address")
		}
	}
	return u, nil
}

func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsMulticast() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 0 || (ip4[0] == 169 && ip4[1] == 254) {
			return false
		}
	}
	return true
}
