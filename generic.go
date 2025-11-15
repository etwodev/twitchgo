package twitchgo

type Method string
type Status string

const (
	Webhook   Method = "webhook"
	Websocket Method = "websocket"

	Enabled  Status = "enabled"
	Disabled Status = "disabled"
)
