package middleware

import (
	"errors"
	"strings"
)

// CheckTokenConfigured fails when AUTH_REQUIRED is true but the API token is blank.
// APIToken leaves the API open with a blank token so it could ship before the token existed;
// once AUTH_REQUIRED=true (nothing else guards the API) that is a misconfiguration, and the
// service should refuse to start rather than serve an open API.
func CheckTokenConfigured(authRequired, token string) error {
	if strings.EqualFold(strings.TrimSpace(authRequired), "true") && strings.TrimSpace(token) == "" {
		return errors.New("AUTH_REQUIRED=true but API_TOKEN is empty: refusing to start with an open API")
	}
	return nil
}
