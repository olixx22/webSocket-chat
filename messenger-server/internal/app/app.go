package app

import (
	"context"
	"log/slog"
	websocketapp "ws_chat/messenger-server/internal/app/websocket"
	"ws_chat/messenger-server/internal/config"
)

type App struct {
	WebSocketApp *websocketapp.App
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger, dbURL string) *App {
	//TODO: implement storage

	webSocketApp := websocketapp.New(cfg, log)

	return &App{
		WebSocketApp: webSocketApp,
	}
}