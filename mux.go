package twitchgo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	c "github.com/Etwodev/twitchgo/pkg/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// handler creates and returns the root chi.Mux router for the server.
//
// It initializes the mux with middleware and routers previously loaded.
//
// Example:
//
//	mux := srv.handler()
func (b *Bot) handler() *chi.Mux {
	m := chi.NewMux()
	b.initMux(m)
	return m
}

// initMux initializes the chi router with routes and middleware.
// It registers the EventSub webhook endpoint using the provided EventEngine.
func (b *Bot) initMux(m *chi.Mux) {
	m.Use(middleware.RequestID)
	m.Use(middleware.RealIP)
	m.Use(middleware.Recoverer)

	m.Route("/webhook", func(r chi.Router) {
		r.Post("/callback", b.webhook.Handle)
	})

	m.Get("/auth/login", b.handleAuthLogin)
	m.With().Get("/auth/callback",
		simpleBasicAuth(
			os.Getenv("CALLBACK_USER"),
			os.Getenv("CALLBACK_PASS"),
			b.handleAuthCallback,
		),
	)

	m.Get("/healthcheck", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			b.logger.Fatal().Str("Function", "initMux").Err(err)
		}
	})
}

func (b *Bot) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s",
		url.QueryEscape(c.ClientID()),
		url.QueryEscape(c.RedirectUri()),
		url.QueryEscape(strings.Join(c.Scopes(), " ")),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (b *Bot) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	data := url.Values{}
	data.Set("client_id", c.ClientID())
	data.Set("client_secret", b.clientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", c.RedirectUri())

	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b.helix.SetUserAccessToken(body.AccessToken)
	b.helix.SetRefreshToken(body.RefreshToken)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Access token stored successfully. Expires in %d seconds.", body.ExpiresIn)
}

// simpleBasicAuth wraps a handler with basic auth protection
func simpleBasicAuth(username, password string, next http.HandlerFunc) http.HandlerFunc {
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
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
