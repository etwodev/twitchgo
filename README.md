# **twitchgo**

`twitchgo` is an event-driven Twitch EventSub bot framework built on Go, providing a structured HTTP server, OAuth2 authentication flow, signature-verified webhook handling, and a pluggable event engine. The framework manages configuration, logging, deduplication of Twitch messages, and graceful shutdown routines.

This project exposes an internal OAuth login flow, accessible at **`/auth/login`**, which generates a bot access token based on your configured scopes.

## **Features**

* Event-driven architecture via a user-defined `EventEngine`
* Webhook processing for Twitch EventSub (signature verification, HMAC validation, replay protection)
* OAuth2 authorization flow for acquiring bot tokens
* Configurable HTTP server with TLS support
* Structured logging using `zerolog`
* Dedupe cache for EventSub message IDs
* Automatic configuration file creation on first run
* Extensible routing via Chi

## **Installation**

```bash
go get github.com/Etwodev/twitchgo
```

## **Quick Start**

### **1. Implement an EventEngine**

Your engine must implement the required callbacks—for example:

```go
type MyEngine struct{}

func (e *MyEngine) OnChannelChatMessage(ctx context.Context, h *helix.Client, event twitchgo.Response[helix.EventSubChannelChatMessageEvent, helix.EventSubCondition]) {
    // handle chat message event
}
```

### **2. Initialize and start the bot**

```go
engine := &MyEngine{}
bot := twitchgo.New(engine)
bot.Start()
```


## **OAuth Flow**

`twitchgo` exposes these endpoints:

### **`GET /auth/login`**

Redirects the user to Twitch OAuth using:

* Client ID from config
* Scopes defined in config (`scopes`)
* Redirect URI from config (`redirectUri`)

### **`GET /auth/callback`**

Twitch redirects back to this endpoint after the user grants permissions.

This endpoint is protected by **Basic Auth**, requiring:

* `CALLBACK_USER`
* `CALLBACK_PASS`

The callback handler exchanges the authorization code for an access token and stores it in your engine or environment as needed.

## **Webhook Handling**

All EventSub notifications are sent to:

### **`POST /webhook/callback`**

This handler performs:

* Required header validation
* Timestamp freshness check
* HMAC signature verification (`CLIENT_SECRET`)
* Duplicate message detection
* Challenge handling
* Dispatch of notifications to your configured `EventEngine`

Supported events include:

* `channel.chat.message` (v1)
  Additional types may require extending `processNotification` with more mappings.

## **Health Check**

### **`GET /healthcheck`**

Returns a simple HTTP 200 response for readiness/liveness checks.

# **Configuration**

`twitchgo` loads configuration from:

```
./twitchgo.config.json
```

This file is created automatically on first run if not present.

### **Example default config**

```json
{
  "port": "7000",
  "address": "0.0.0.0",
  "experimental": false,
  "readTimeout": 15,
  "writeTimeout": 15,
  "idleTimeout": 60,
  "logLevel": "info",
  "maxHeaderBytes": 1048576,
  "enableTLS": false,
  "tlsCertFile": "",
  "tlsKeyFile": "",
  "shutdownTimeout": 15,
  "enableCORS": false,
  "allowedOrigins": [],
  "enableRequestLogging": false,
  "scopes": ["channel:moderate"],
  "redirectUri": "https://example.com",
  "clientId": "unknown"
}
```

### **Config fields**

| Field                                          | Description                               |
| ---------------------------------------------- | ----------------------------------------- |
| `port`                                         | HTTP server port                          |
| `address`                                      | Bind address                              |
| `experimental`                                 | Enables experimental middleware/endpoints |
| `readTimeout` / `writeTimeout` / `idleTimeout` | Request timeouts (seconds)                |
| `logLevel`                                     | Logging level (`debug`, `info`, etc.)     |
| `maxHeaderBytes`                               | Maximum request header size               |
| `enableTLS`                                    | Enables HTTPS server                      |
| `tlsCertFile` / `tlsKeyFile`                   | Certificate and key paths                 |
| `shutdownTimeout`                              | Graceful shutdown timeout (seconds)       |
| `enableCORS`                                   | Enables CORS middleware                   |
| `allowedOrigins`                               | Allowed CORS origins                      |
| `enableRequestLogging`                         | Enables request logging middleware        |
| `scopes`                                       | Twitch OAuth scopes                       |
| `redirectUri`                                  | OAuth redirect URL                        |
| `clientId`                                     | Twitch client ID                          |


# **Required Environment Variables**

| Variable        | Purpose                                                             |
| --------------- | ------------------------------------------------------------------- |
| `CLIENT_SECRET` | Twitch application client secret used for OAuth and HMAC validation |
| `CALLBACK_USER` | Username for callback Basic Auth                                    |
| `CALLBACK_PASS` | Password for callback Basic Auth                                    |

All OAuth and signature verification processes depend on these being set.

# **Running the Server**

### **Local**

```bash
export CLIENT_SECRET="your_twitch_secret"
export CALLBACK_USER="admin"
export CALLBACK_PASS="supersecret"

go run main.go
```

Server starts at:

```
http://0.0.0.0:7000
```

### **With TLS**

Enable TLS in config:

```json
"enableTLS": true,
"tlsCertFile": "cert.pem",
"tlsKeyFile": "key.pem"
```

# **Event Subscription**

Subscriptions are created externally using the Twitch Helix API.
You may use the built-in `helix.Client` provided via:

```go
bot.Helix()
```

Ensure your EventSub transport points to:

```
https://<your-domain>/webhook/callback
```


# **Extending Event Types**

To support additional Twitch EventSub notifications:

1. Add your type mapping inside `processNotification`
2. Unmarshal into a `Response[...]`
3. Dispatch to your engine:

```go
go b.engine.OnYourEvent(ctx, b.helix, event)
```

# **Graceful Shutdown**

The bot listens for `os.Interrupt` and shuts down the server cleanly, respecting the configured `shutdownTimeout`.


# **Logging**

`twitchgo` uses `zerolog` with a console-friendly writer.
Global log level is set via configuration.

# **License**

MIT (recommended to update with your own license if applicable).
