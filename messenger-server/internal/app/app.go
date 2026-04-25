package app

import (
	"context"
	"fmt"
	"log/slog"
	websocketapp "ws_chat/messenger-server/internal/app/websocket"
	"ws_chat/messenger-server/internal/config"
	"ws_chat/messenger-server/internal/storage/postgres"
)

type App struct {
	WebSocketApp *websocketapp.App
	Storage      *postgres.Storage
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger, dbURL string) (*App, error) {
	const op = "app.New"

	db, err := postgres.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	webSocketApp := websocketapp.New(cfg, log, db)

	return &App{
		WebSocketApp: webSocketApp,
		Storage:      db,
	}, nil
}

func (a *App) Stop(ctx context.Context) error {
	if err := a.WebSocketApp.Stop(ctx); err != nil {
		return err
	}

	if a.Storage != nil {
		if err := a.Storage.Close(ctx); err != nil {
			return err
		}
	}

	return nil
}
