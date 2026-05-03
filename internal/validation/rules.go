package validation

import (
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
)

func registerCustom(v *validator.Validate) {
	v.RegisterValidation("domain_or_url", domainOrURL)
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
