package validation

import (
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
)

func registerCustom(v *validator.Validate) {
	v.RegisterValidation("domain_or_url", domainOrURL)
	v.RegisterValidation("username", username)
}

func domainOrURL(fl validator.FieldLevel) bool {
	val := strings.TrimSpace(fl.Field().String())
	if val == "" {
		return false
	}

	// Case 1: full URL
	if strings.Contains(val, "://") {
		u, err := url.ParseRequestURI(val)
		if err != nil {
			return false
		}
		return u.Scheme == "http" || u.Scheme == "https"
	}

	// Case 2: bare domain
	if strings.Contains(val, ".") && !strings.Contains(val, " ") {
		return true
	}

	return false
}

// username allows stable account handles: letters, numbers, dot, dash, and underscore only.
func username(fl validator.FieldLevel) bool {
	val := strings.TrimSpace(fl.Field().String())
	if val != fl.Field().String() {
		return false
	}

	for _, r := range val {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}

	return true
}
