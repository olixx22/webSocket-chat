package websocketapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/config"
	MWLogger "ws_chat/messenger-server/internal/websocket-server/middleware/logger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

type App struct {
	Router *chi.Mux
	Server *http.Server
	WsUpgrader *websocket.Upgrader 
	Log *slog.Logger
}

func New(cfg *config.Config, log *slog.Logger) *App {
	router := chi.NewRouter()

	setupMiddlewares(router, log)

	srv := &http.Server{
		Addr: cfg.WSServer.Address,
		Handler: router,
		ReadTimeout: cfg.WSServer.Timeout,
		WriteTimeout: cfg.WSServer.Timeout,
		IdleTimeout: cfg.WSServer.IdleTimeout,
	}

	wsUpg := &websocket.Upgrader{}

	return &App{
		Router: router,
		Server: srv,
		WsUpgrader: wsUpg,
		Log: log,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "websocketapp.Run"

	log := a.Log.With(slog.String("op", op))

	log.Info("starting http and websocket server")

	if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed{
		log.Error("failed to start server")
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop(ctx context.Context) error {
	const op = "websocketapp.Stop"

	if err := a.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}


func setupMiddlewares(router *chi.Mux, log *slog.Logger) {
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(MWLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	//...add more middlewares here if needed
}