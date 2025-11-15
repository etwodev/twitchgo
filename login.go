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
	"analytics:read:extensions",
	"analytics:read:games",
	"bits:read",
	"channel:bot",
	"channel:manage:ads",
	"channel:read:ads",
	"channel:manage:broadcast",
	"channel:read:charity",
	"channel:manage:clips",
	"channel:edit:commercial",
	"channel:read:editors",
	"channel:manage:extensions",
	"channel:read:goals",
	"channel:read:guest_star",
	"channel:manage:guest_star",
	"channel:read:hype_train",
	"channel:manage:moderators",
	"channel:read:polls",
	"channel:manage:polls",
	"channel:read:predictions",
	"channel:manage:predictions",
	"channel:manage:raids",
	"channel:read:redemptions",
	"channel:manage:redemptions",
	"channel:manage:schedule",
	"channel:read:stream_key",
	"channel:read:subscriptions",
	"channel:manage:videos",
	"channel:read:vips",
	"channel:manage:vips",
	"channel:moderate",
	"clips:edit",
	"editor:manage:clips",
	"moderation:read",
	"moderator:manage:banned_users",
	"moderator:read:blocked_terms",
	"moderator:read:chat_messages",
	"moderator:manage:blocked_terms",
	"moderator:manage:chat_messages",
	"moderator:read:chat_settings",
	"moderator:manage:chat_settings",
	"moderator:read:chatters",
	"moderator:read:followers",
	"moderator:read:guest_star",
	"moderator:manage:guest_star",
	"moderator:read:shield_mode",
	"moderator:manage:shield_mode",
	"moderator:read:shoutouts",
	"moderator:manage:shoutouts",
	"moderator:read:suspicious_users",
	"moderator:read:unban_requests",
	"moderator:manage:unban_requests",
	"moderator:read:vips",
	"moderator:read:warnings",
	"moderator:manage:warnings",
	"user:bot",
	"user:edit",
	"user:edit:broadcast",
	"user:read:blocked_users",
	"user:manage:blocked_users",
	"user:read:broadcast",
	"user:read:chat",
	"user:manage:chat_color",
	"user:read:email",
	"user:read:emotes",
	"user:read:follows",
	"user:read:moderated_channels",
	"user:read:subscriptions",
	"user:read:whispers",
	"user:manage:whispers",
	"user:write:chat",
	"chat:edit",
	"chat:read",
	"whispers:read",
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
