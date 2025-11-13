package twitchgo

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	c "github.com/Etwodev/twitchgo/pkg/config"
	"github.com/Etwodev/twitchgo/pkg/engine"
	"github.com/Etwodev/twitchgo/pkg/log"
	"github.com/nicklaw5/helix/v2"
	"github.com/rs/zerolog"
)

// Server represents an HTTP server with support for
// configuration, middleware, routers, and structured logging.
type Bot struct {
	clientSecret string
	logger       log.Logger
	instance     *http.Server
	helix        *helix.Client
	idle         chan struct{}
	webhook      *engine.WebhookClient
}

// New creates a new Bot instance with configuration loaded
// and a logger initialized.
//
// It will fatal exit if configuration loading failb.
//
// Example:
//
//	bot := twitchgo.New()
func New(eng engine.EventEngine, clientSecret string) *Bot {
	err := c.New()
	if err != nil {
		baseLogger := zerolog.New(os.Stdout).With().Timestamp().Str("Group", "twitchgo").Logger()
		baseLogger.Fatal().Str("Function", "New").Err(err).Msg("Failed to load config")
	}

	level, err := zerolog.ParseLevel(c.LogLevel())
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	format := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05"}
	baseLogger := zerolog.New(format).With().Timestamp().Str("Group", "twitchgo").Logger()

	logger := log.NewZeroLogger(baseLogger)

	opts := &helix.Options{
		ClientID:     c.ClientID(),
		ClientSecret: clientSecret,
	}

	client, err := helix.NewClient(opts)
	if err != nil {
		logger.Fatal().Str("Function", "New").Err(err).Msg("Failed to setup helix client")
	}

	webhook := engine.NewWebHookClient(eng, clientSecret)

	return &Bot{
		logger:       logger,
		helix:        client,
		webhook:      webhook,
		clientSecret: clientSecret,
	}
}

// Logger returns the logger instance used by the bot.
//
// Example:
//
//	logger := bot.Logger()
//	logger.Info().Msg("Bot logger retrieved")
func (b *Bot) Logger() log.Logger {
	return b.logger
}

// Helix returns the helix client instance used by the bot.
//
// Example
//
// helix := bot.Helix()
// helix.

// Start launches the HTTP server, applying configured middleware and routers,
// and listens for termination signals for graceful shutdown.
//
// It blocks until the server is shut down.
//
// Example:
//
//	bot.Start()
func (b *Bot) Start() {
	b.instance = &http.Server{
		Addr:           fmt.Sprintf("%s:%s", c.Address(), c.Port()),
		Handler:        b.handler(),
		ReadTimeout:    time.Duration(c.ReadTimeout()) * time.Second,
		WriteTimeout:   time.Duration(c.WriteTimeout()) * time.Second,
		IdleTimeout:    time.Duration(c.IdleTimeout()) * time.Second,
		MaxHeaderBytes: c.MaxHeaderBytes(),
	}

	b.logger.Debug().
		Str("Port", c.Port()).
		Str("Address", c.Address()).
		Bool("Experimental", c.Experimental()).
		Msg("Server started")

	b.idle = make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		timeout := time.Duration(c.ShutdownTimeout()) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := b.instance.Shutdown(ctx); err != nil {
			b.logger.Warn().Str("Function", "Shutdown").Err(err).Msg("Server shutdown failed!")
		}
		close(b.idle)
	}()

	if c.EnableTLS() {
		b.logger.Info().Msg("Starting HTTPS server")
		if err := b.instance.ListenAndServeTLS(c.TLSCertFile(), c.TLSKeyFile()); err != nil && err != http.ErrServerClosed {
			b.logger.Fatal().Err(err).Msg("HTTPS server failed")
		}
	} else {
		b.logger.Info().Msg("Starting HTTP server")
		if err := b.instance.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			b.logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}

	<-b.idle

	b.logger.Debug().
		Str("Port", c.Port()).
		Str("Address", c.Address()).
		Bool("Experimental", c.Experimental()).
		Msg("Server stopped")
}
