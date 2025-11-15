package twitchgo

import (
	"net/http"
	"path"

	c "github.com/Etwodev/twitchgo/pkg/config"
	"github.com/Etwodev/twitchgo/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

// handler creates and returns the root chi.Mux router for the server.
//
// It initializes the mux with middleware and routers previously loaded.
//
// Example:
//
//	mux := srv.handler()
func (b *Bot) handler() *chi.Mux {
	m := chi.NewMux()
	b.Routes(m)
	b.initMux(m)
	return m
}

func (b *Bot) initMux(m *chi.Mux) {
	if c.EnableRequestLogging() {
		middleware := middleware.NewLoggingMiddleware(b.logger)

		b.logger.Debug().
			Str("Name", middleware.Name()).
			Bool("Experimental", middleware.Experimental()).
			Bool("Status", middleware.Status()).
			Msg("Registering middleware")

		m.Use(middleware.Method())
	}

	if c.EnableCORS() && len(c.AllowedOrigins()) > 0 {
		middleware := middleware.NewCORSMiddleware(c.AllowedOrigins())

		b.logger.Debug().
			Str("Name", middleware.Name()).
			Bool("Experimental", middleware.Experimental()).
			Bool("Status", middleware.Status()).
			Msg("Registering middleware")

		m.Use(middleware.Method())
	}

	for _, middleware := range b.middlewares {
		if middleware.Status() && (middleware.Experimental() == c.Experimental() || !middleware.Experimental()) {
			b.logger.Debug().
				Str("Name", middleware.Name()).
				Bool("Experimental", middleware.Experimental()).
				Bool("Status", middleware.Status()).
				Msg("Registering middleware")

			m.Use(middleware.Method())
		}
	}

	for _, rtr := range b.routers {
		if !rtr.Status() {
			continue
		}

		m.Route(rtr.Prefix(), func(r chi.Router) {
			for _, rmw := range rtr.Middleware() {
				r.Use(rmw)
			}

			for _, rt := range rtr.Routes() {
				if !rt.Status() || (rt.Experimental() != c.Experimental() && rt.Experimental()) {
					continue
				}

				b.logger.Debug().
					Bool("Experimental", rt.Experimental()).
					Bool("Status", rt.Status()).
					Str("Method", rt.Method()).
					Str("Path", path.Join(rtr.Prefix(), rt.Path())).
					Msg("Registering route")

				finalHandler := http.Handler(rt.Handler())
				for _, mw := range rt.Middleware() {
					finalHandler = mw(finalHandler)
				}

				r.Method(rt.Method(), rt.Path(), finalHandler)
			}
		})
	}
}
