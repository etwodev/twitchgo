package twitchgo

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Etwodev/twitchgo/pkg/config"
	"github.com/Etwodev/twitchgo/pkg/log"
	"github.com/Etwodev/twitchgo/pkg/middleware"
	"github.com/Etwodev/twitchgo/pkg/router"
	"github.com/nicklaw5/helix/v2"
	"github.com/rs/zerolog"
)

// configuration, middleware, routers, and structured logging.
type Bot struct {
	logger      log.Logger
	engine      EventEngine
	cache       *dedupeCache
	instance    *http.Server
	helix       *helix.Client
	middlewares []middleware.Middleware
	routers     []router.Router
	idle        chan struct{}
}

// LoadRouter appends one or more routers to the server's router list.
//
// Example:
//
//	srv.LoadRouter([]router.Router{myRouter1, myRouter2})
func (b *Bot) LoadRouter(routers []router.Router) {
	b.routers = append(b.routers, routers...)
}

// LoadMiddleware appends one or more middleware instances to the server's middleware chain.
//
// Middleware registered here will be applied globally to all routers.
//
// Example:
//
//	srv.LoadMiddleware([]middleware.Middleware{corsMw, loggingMw})
func (b *Bot) LoadMiddleware(middlewares []middleware.Middleware) {
	b.middlewares = append(b.middlewares, middlewares...)
}

// New creates a new Bot instance with configuration loaded
// and a logger initialized.
//
// It will fatal exit if configuration loading failb.
//
// Example:
//
//	bot := twitchgo.New()
func New(engine EventEngine) *Bot {
	err := config.New()
	if err != nil {
		baseLogger := zerolog.New(os.Stdout).With().Timestamp().Str("Group", "twitchgo").Logger()
		baseLogger.Fatal().Str("Function", "New").Err(err).Msg("Failed to load config")
	}

	level, err := zerolog.ParseLevel(config.LogLevel())
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	format := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05"}
	baseLogger := zerolog.New(format).With().Timestamp().Str("Group", "twitchgo").Logger()
	logger := log.NewZeroLogger(baseLogger)

	transport := &HelixRefreshTransport{
		Base:  http.DefaultTransport,
		Event: engine,
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	opts := &helix.Options{
		HTTPClient:   httpClient,
		ClientID:     config.ClientID(),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
	}

	client, err := helix.NewClient(opts)
	if err != nil {
		logger.Fatal().Str("Function", "New").Err(err).Msg("Failed to setup helix client")
	}

	transport.Client = client

	return &Bot{
		engine: engine,
		logger: logger,
		helix:  client,
		cache:  newDedupeCache(5 * time.Minute),
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

// Helix returns the helix instance used by the bot.
//
// Example:
//
//	helix := bot.Helix()
//	helix.GetUser...
func (b *Bot) Helix() *helix.Client {
	return b.helix
}

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
		Addr:           fmt.Sprintf("%s:%s", config.Address(), config.Port()),
		Handler:        b.handler(),
		ReadTimeout:    time.Duration(config.ReadTimeout()) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout()) * time.Second,
		IdleTimeout:    time.Duration(config.IdleTimeout()) * time.Second,
		MaxHeaderBytes: config.MaxHeaderBytes(),
	}

	b.logger.Debug().
		Str("Port", config.Port()).
		Str("Address", config.Address()).
		Bool("Experimental", config.Experimental()).
		Msg("Server starting")

	// NOTE: Investigate what sort of context should be used here
	b.engine.OnBotStart(context.Background(), b.helix)

	b.idle = make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		timeout := time.Duration(config.ShutdownTimeout()) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := b.instance.Shutdown(ctx); err != nil {
			b.logger.Warn().Str("Function", "Shutdown").Err(err).Msg("Server shutdown failed!")
		}
		close(b.idle)
	}()

	if config.EnableTLS() {
		b.logger.Info().Msg("Starting HTTPS server")
		if err := b.instance.ListenAndServeTLS(config.TLSCertFile(), config.TLSKeyFile()); err != nil && err != http.ErrServerClosed {
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
		Str("Port", config.Port()).
		Str("Address", config.Address()).
		Bool("Experimental", config.Experimental()).
		Msg("Server stopped")
}
