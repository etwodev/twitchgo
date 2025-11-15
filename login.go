package twitchgo

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Etwodev/twitchgo/pkg/config"
	"github.com/Etwodev/twitchgo/pkg/helpers"
)

var FULL_AUTH_SCOPES = []string{
	"user:read:email",
	"user:read:subscriptions",
	"user:edit",
	"user:edit:follows",
	"channel:read:subscriptions",
	"channel:read:redemptions",
	"channel:manage:redemptions",
	"channel:read:editors",
	"channel:manage:videos",
	"channel:moderate",
	"chat:read",
	"chat:edit",
	"moderation:read",
	"moderation:manage",
	"whispers:read",
	"whispers:edit",
	"clips:edit",
	"analytics:read:games",
	"bits:read",
	"channel:read:hype_train",
	"channel:manage:broadcast",
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := helpers.GenerateState(24)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     "twitch_oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   config.EnableTLS(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(10 * time.Minute),
	}
	http.SetCookie(w, cookie)

	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		url.QueryEscape(config.ClientID()),
		url.QueryEscape(config.RedirectUri()),
		url.QueryEscape(strings.Join(FULL_AUTH_SCOPES, " ")),
		url.QueryEscape(state),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}
