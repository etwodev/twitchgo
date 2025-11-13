package engine

import (
	"context"

	"github.com/nicklaw5/helix/v2"
)

type EventEngine interface {
	OnChannelChatMessage(ctx context.Context, response Response[helix.EventSubChannelChatMessageEvent, helix.EventSubCondition])
}
