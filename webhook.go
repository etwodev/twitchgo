package twitchgo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nicklaw5/helix/v2"
)

func (b *Bot) Handle(w http.ResponseWriter, r *http.Request) {
	msgID := r.Header.Get("Twitch-Eventsub-Message-Id")
	msgType := r.Header.Get("Twitch-Eventsub-Message-Type")
	msgTS := r.Header.Get("Twitch-Eventsub-Message-Timestamp")
	msgSig := r.Header.Get("Twitch-Eventsub-Message-Signature")

	if msgID == "" || msgType == "" || msgSig == "" || msgTS == "" {
		http.Error(w, "missing headers", http.StatusBadRequest)
		return
	}

	if t, err := time.Parse(time.RFC3339, msgTS); err != nil || time.Since(t) > 10*time.Minute {
		http.Error(w, "invalid or expired timestamp", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	msg := BuildHMACMessage(msgID, msgTS, body)
	computed := ComputeHMAC([]byte(os.Getenv("CLIENT_SECRET")), msg)
	if !VerifyHMAC(computed, msgSig) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	if b.cache.Exists(msgID) {
		w.WriteHeader(http.StatusOK)
		return
	}

	b.cache.Add(msgID)
	switch msgType {
	case "notification":
		err := processNotification(r.Context(), body, b)
		if err != nil {
			http.Error(w, "failed to process notification", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	case "webhook_callback_verification":
		resp, err := handleChallenge(body)
		if err != nil {
			http.Error(w, "failed challenge", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	case "revocation":
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "unknown message type", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// processNotification processes a given notification for a given body
func processNotification(ctx context.Context, body []byte, b *Bot) error {
	var wrapper struct {
		Subscription Subscription[any] `json:"subscription"`
		Event        json.RawMessage   `json:"event"`
	}

	if err := json.Unmarshal(body, &wrapper); err != nil {
		return err
	}

	switch strings.ToLower(wrapper.Subscription.Type + ".v" + wrapper.Subscription.Version) {
	case string(ChannelChatMessage):
		var event Response[helix.EventSubChannelChatMessageEvent, helix.EventSubCondition]
		if err := json.Unmarshal(body, &event); err != nil {
			return err
		}

		go b.engine.OnChannelChatMessage(ctx, b.helix, event)
		return nil
	default:
		return fmt.Errorf("unsupported subscription type: %s", wrapper.Subscription.Type)
	}
}
