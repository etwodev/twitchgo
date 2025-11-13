package engine

import "time"

// Request represents an EventSub subscription response.
//
// See: https://dev.twitch.tv/docs/eventsub/eventsub-reference for more information.
type Response[T interface{}, U interface{}] struct {
	Event        T               `json:"event"`
	Subscription Subscription[U] `json:"subscription"`
}

type Subscription[T interface{}] struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	Status    Status    `json:"status"`
	Cost      int       `json:"cost"`
	Condition T         `json:"condition"`
	CreatedAt time.Time `json:"created_at"`
}
