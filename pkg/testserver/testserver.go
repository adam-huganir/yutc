//go:build testservercmd

package testserver

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
)

type AuthConfig struct {
	BasicUser   string
	BasicPass   string
	BearerToken string
}

func ValidateAuthConfig(cfg AuthConfig) error {
	if cfg.BasicUser == "" && cfg.BasicPass == "" && cfg.BearerToken == "" {
		return fmt.Errorf("no auth configured")
	}
	if (cfg.BasicUser == "") != (cfg.BasicPass == "") {
		return fmt.Errorf("basic auth requires both user and pass")
	}
	return nil
}

func Handler(root string, cfg AuthConfig) (http.Handler, error) {
	if err := ValidateAuthConfig(cfg); err != nil {
		return nil, err
	}
	fs := http.FileServer(http.Dir(root))
	return withAuth(fs, cfg), nil
}

func withAuth(next http.Handler, cfg AuthConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authorized(r, cfg) {
			next.ServeHTTP(w, r)
			return
		}
		if cfg.BasicUser != "" && cfg.BasicPass != "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="test-server"`)
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func authorized(r *http.Request, cfg AuthConfig) bool {
	allowBasic := cfg.BasicUser != "" && cfg.BasicPass != ""
	allowBearer := cfg.BearerToken != ""

	if allowBasic {
		user, pass, ok := r.BasicAuth()
		if ok && secureEqual(user, cfg.BasicUser) && secureEqual(pass, cfg.BasicPass) {
			return true
		}
	}

	if allowBearer {
		authz := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if strings.HasPrefix(authz, prefix) {
			token := strings.TrimPrefix(authz, prefix)
			if secureEqual(token, cfg.BearerToken) {
				return true
			}
		}
	}

	return false
}

func secureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
