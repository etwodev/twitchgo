package twitchgo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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
	m.Get("/auth/callback", simpleBasicAuth(
		os.Getenv("CALLBACK_USER"),
		os.Getenv("CALLBACK_PASS"),
		b.handleAuthCallback,
	))

	m.Get("/healthcheck", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			b.logger.Fatal().Str("Function", "initMux").Err(err)
		}
	})
}

// handleAuthLogin starts the OAuth Authorization Code flow with a state nonce saved in a secure cookie.
func (b *Bot) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateState(24)
	if err != nil {
		b.logger.Error().Err(err).Msg("failed to generate oauth state")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     "twitch_oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.EnableTLS(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(10 * time.Minute),
	}
	http.SetCookie(w, cookie)

	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		url.QueryEscape(c.ClientID()),
		url.QueryEscape(c.RedirectUri()),
		url.QueryEscape(strings.Join(c.Scopes(), " ")),
		url.QueryEscape(state),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleAuthCallback handles the token exchange and sets the tokens on the helix client.
// It verifies the state value and persists the refresh token via TokenStore.
func (b *Bot) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("twitch_oauth_state")
	if err != nil || cookie.Value == "" || cookie.Value != state {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read token response", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		b.logger.Error().
			Int("status", resp.StatusCode).
			Msg(fmt.Sprintf("token endpoint returned non-200: %s", string(bodyBytes)))
		http.Error(w, "token endpoint error", http.StatusInternalServerError)
		return
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		b.logger.Error().Err(err).Msg("failed to decode token response")
		http.Error(w, "invalid token response", http.StatusInternalServerError)
		return
	}

	if body.AccessToken != "" {
		b.helix.SetUserAccessToken(body.AccessToken)
	}

	if body.RefreshToken != "" {
		b.helix.SetRefreshToken(body.RefreshToken)
	}

	b.engine.OnTokenVerify()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("Access token stored successfully. Expires in %d seconds.", body.ExpiresIn)))
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
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
