package helpers

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// SimpleBasicAuth wraps a handler with basic auth protection
func SimpleBasicAuth(username, password string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Basic ") {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || pair[0] != username || pair[1] != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
