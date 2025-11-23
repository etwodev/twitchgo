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
	b.logger.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Msg("received EventSub request")

	msgID := r.Header.Get("Twitch-Eventsub-Message-Id")
	msgType := r.Header.Get("Twitch-Eventsub-Message-Type")
	msgTS := r.Header.Get("Twitch-Eventsub-Message-Timestamp")
	msgSig := r.Header.Get("Twitch-Eventsub-Message-Signature")

	b.logger.Debug().
		Str("message_id", msgID).
		Str("message_type", msgType).
		Str("timestamp", msgTS).
		Str("signature", msgSig).
		Msg("extracted headers")

	if msgID == "" || msgType == "" || msgSig == "" || msgTS == "" {
		b.logger.Warn().Msg("missing required headers")
		http.Error(w, "missing headers", http.StatusBadRequest)
		return
	}

	t, err := time.Parse(time.RFC3339, msgTS)
	if err != nil {
		b.logger.Warn().Err(err).Msg("failed to parse message timestamp")
		http.Error(w, "invalid timestamp", http.StatusBadRequest)
		return
	}
	if time.Since(t) > 10*time.Minute {
		b.logger.Warn().Int("timestamp", int(t.Unix())).Msg("timestamp expired")
		http.Error(w, "expired timestamp", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		b.logger.Error().Err(err).Msg("failed to read request body")
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	b.logger.Debug().Int("body_bytes", len(body)).Msg("read request body")

	msg := BuildHMACMessage(msgID, msgTS, body)
	computed := ComputeHMAC([]byte(os.Getenv("CLIENT_SECRET")), msg)

	if !VerifyHMAC(computed, msgSig) {
		b.logger.Warn().Msg("signature verification failed")
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	b.logger.Debug().Msg("HMAC signature verified")

	if b.cache.Exists(msgID) {
		b.logger.Debug().Str("message_id", msgID).Msg("duplicate message; already processed")
		w.WriteHeader(http.StatusOK)
		return
	}

	b.cache.Add(msgID)
	b.logger.Debug().Str("message_id", msgID).Msg("cached message ID")

	switch msgType {
	case "notification":
		b.logger.Debug().Msg("handling notification")
		if err := processNotification(r.Context(), body, b); err != nil {
			b.logger.Error().Err(err).Msg("failed to process notification")
			http.Error(w, "failed to process notification", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "webhook_callback_verification":
		b.logger.Debug().Msg("handling callback verification challenge")
		resp, err := handleChallenge(body)
		if err != nil {
			b.logger.Error().Err(err).Msg("failed to process challenge")
			http.Error(w, "failed challenge", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)

	case "revocation":
		b.logger.Warn().Msg("received subscription revocation")
		w.WriteHeader(http.StatusOK)

	default:
		b.logger.Warn().Str("message_type", msgType).Msg("unknown message type")
		http.Error(w, "unknown message type", http.StatusBadRequest)
		return
	}
}

// processNotification processes a given notification
func processNotification(ctx context.Context, body []byte, b *Bot) error {
	b.logger.Debug().Msg("processing notification wrapper")

	var wrapper struct {
		Subscription Subscription[any] `json:"subscription"`
		Event        json.RawMessage   `json:"event"`
	}

	if err := json.Unmarshal(body, &wrapper); err != nil {
		b.logger.Error().Err(err).Msg("failed to unmarshal notification wrapper")
		return err
	}

	subKey := strings.ToLower(wrapper.Subscription.Type + ".v" + wrapper.Subscription.Version)
	b.logger.Debug().Str("subscription", subKey).Msg("parsed subscription type")

	switch subKey {
	case string(ChannelChatMessage):
		var event Response[helix.EventSubChannelChatMessageEvent, helix.EventSubCondition]
		if err := json.Unmarshal(body, &event); err != nil {
			b.logger.Error().Err(err).Msg("failed to unmarshal chat message event")
			return err
		}

		b.logger.Debug().
			Str("broadcaster_id", event.Event.BroadcasterUserID).
			Str("user_id", event.Event.ChatterUserID).
			Msg("dispatching channel chat message handler")

		go b.engine.OnChannelChatMessage(ctx, b.helix, event)
		return nil

	default:
		err := fmt.Errorf("unsupported subscription type: %s", wrapper.Subscription.Type)
		b.logger.Warn().Err(err).Msg("unsupported subscription received")
		return err
	}
}
