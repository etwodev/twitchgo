package twitchgo

import (
	"context"

	"github.com/nicklaw5/helix/v2"
)

type SubscriptionType string

const (
	ChannelChatMessage SubscriptionType = "channel.chat.message.v1"
)

// EventEngine defines an interface for handling various Twitch bot events.
type EventEngine interface {
	// OnBotStart is called when the bot starts.
	// Useful for initializing connections, registering event subscriptions, or performing startup routines.
	OnBotStart(ctx context.Context, api *helix.Client)

	// OnClientLogin is called when a user logs into the client.
	// At this point, access and refresh tokens are set, allowing API calls on behalf of the user.
	OnClientLogin(ctx context.Context, api *helix.Client)

	// OnClientRefresh is called whenever an access token is refreshed.
	// This ensures the bot continues to operate with a valid token without interruption.
	OnClientRefresh(ctx context.Context, api *helix.Client)

	
	OnChannelChatMessage(ctx context.Context, api *helix.Client, response Response[helix.EventSubChannelChatMessageEvent, helix.EventSubCondition])
}
