package services

import (
	"net"
	"net/url"
	"strings"
)

// NormalizeProjectHost converts domains, URLs, and Host headers into the same
// lowercase host key used for project-domain lookups.
func NormalizeProjectHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}

	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil {
			host = u.Host
		}
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	return strings.Trim(host, ".")
}
