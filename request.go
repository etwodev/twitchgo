package twitchgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nicklaw5/helix/v2"
)

// Request represents an EventSub subscription request.
//
// See: https://dev.twitch.tv/docs/eventsub/eventsub-reference for more information.
type Request[T any] struct {
	Type      SubscriptionType        `json:"type"`
	Version   string                  `json:"version"`
	Condition T                       `json:"condition"`
	Transport helix.EventSubTransport `json:"transport"`
}

func CreateEventSubSubscription[T any](
	ctx context.Context,
	accessToken string,
	request Request[T],
	clientID string,
	client *http.Client,
) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.twitch.tv/helix/eventsub/subscriptions",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
}
