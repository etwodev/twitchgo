package twitchgo

import (
	"context"

	"github.com/nicklaw5/helix/v2"
)

type SubscriptionType string

const (
	ChannelChatMessage SubscriptionType = "channel.chat.message"
)

type EventEngine interface {
	OnBotLogin(ctx context.Context, api *helix.Client)
	OnChannelChatMessage(ctx context.Context, api *helix.Client, response Response[helix.EventSubChannelChatMessageEvent, helix.EventSubCondition])
}
