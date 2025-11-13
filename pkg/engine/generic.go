package engine

type Method string
type Status string
type SubscriptionType string

const (
	Webhook   Method = "webhook"
	Websocket Method = "websocket"

	Enabled  Status = "enabled"
	Disabled Status = "disabled"

	ChannelChatMessage SubscriptionType = "channel.chat.message"
)
