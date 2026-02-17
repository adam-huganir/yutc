package loader

import (
	"crypto/subtle"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testAuthConfig struct {
	basicUser   string
	basicPass   string
	bearerToken string
}

func newAuthFileServer(t *testing.T, cfg testAuthConfig) *httptest.Server {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("unable to get working directory: %v", err)
	}
	root := filepath.Join(wd, "..", "..", "testFiles")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("unable to stat testFiles dir %q: %v", root, err)
	}

	if cfg.basicUser == "" && cfg.basicPass == "" && cfg.bearerToken == "" {
		t.Fatalf("invalid test auth config: no auth configured")
	}
	if (cfg.basicUser == "") != (cfg.basicPass == "") {
		t.Fatalf("invalid test auth config: basic auth requires both user and pass")
	}

	fs := http.FileServer(http.Dir(root))

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if testAuthorized(r, cfg) {
			fs.ServeHTTP(w, r)
			return
		}
		if cfg.basicUser != "" && cfg.basicPass != "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="test-server"`)
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func testAuthorized(r *http.Request, cfg testAuthConfig) bool {
	allowBasic := cfg.basicUser != "" && cfg.basicPass != ""
	allowBearer := cfg.bearerToken != ""

	if allowBasic {
		user, pass, ok := r.BasicAuth()
		if ok && testSecureEqual(user, cfg.basicUser) && testSecureEqual(pass, cfg.basicPass) {
			return true
		}
	}

	if allowBearer {
		authz := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if strings.HasPrefix(authz, prefix) {
			token := strings.TrimPrefix(authz, prefix)
			if testSecureEqual(token, cfg.bearerToken) {
				return true
			}
		}
	}

	return false
}

func testSecureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
