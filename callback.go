package twitchgo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/Etwodev/twitchgo/pkg/config"
)

func (b *Bot) HandleCallback(w http.ResponseWriter, r *http.Request) {
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
	data.Set("client_id", config.ClientID())
	data.Set("client_secret", os.Getenv("CLIENT_SECRET"))
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", config.RedirectUri())

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
		http.Error(w, "token endpoint error", http.StatusInternalServerError)
		return
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		http.Error(w, "invalid token response", http.StatusInternalServerError)
		return
	}

	b.helix.SetUserAccessToken(body.AccessToken)
	b.helix.SetRefreshToken(body.RefreshToken)

	b.engine.OnClientLogin(r.Context(), b.helix)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("Access token stored successfully. Expires in %d seconds.", body.ExpiresIn)))
}
