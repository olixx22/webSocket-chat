package websocketapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/config"
	"ws_chat/messenger-server/internal/storage/postgres"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/addmembers"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/createchat"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/createprivatechat"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/deletechat"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/getchatmembers"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/getuserchat"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/getuserchatswithmessages"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/getuserrole"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/leavechat"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/removemembers"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/chats/updatechat"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/health"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/messages/createmessage"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/messages/deletemessage"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/messages/getmessages"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/messages/getmessagestatus"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/messages/markasread"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/messages/updatemessage"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/users/createuser"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/users/deleteuser"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/users/getuser"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/users/getuserbyname"
	"ws_chat/messenger-server/internal/websocket-server/http-server/handlers/users/updateuser"
	wsconnect "ws_chat/messenger-server/internal/websocket-server/http-server/handlers/ws/connect"
	MWIsChatMember "ws_chat/messenger-server/internal/websocket-server/http-server/middleware/ischatmember"
	MWIsChatOwner "ws_chat/messenger-server/internal/websocket-server/http-server/middleware/ischatowner"
	MWLogger "ws_chat/messenger-server/internal/websocket-server/http-server/middleware/logger"
	"ws_chat/messenger-server/internal/websocket-server/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

type App struct {
	Router     *chi.Mux
	Server     *http.Server
	WsUpgrader *websocket.Upgrader
	Hub        *ws.Hub
	Log        *slog.Logger
	Storage    *postgres.Storage
}

func New(cfg *config.Config, log *slog.Logger, db *postgres.Storage) *App {
	router := chi.NewRouter()

	setupMiddlewares(router, log)

	srv := &http.Server{
		Addr:         cfg.WSServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.WSServer.Timeout,
		WriteTimeout: cfg.WSServer.Timeout,
		IdleTimeout:  cfg.WSServer.IdleTimeout,
	}

	wsUpg := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	hub := ws.NewHub(log)

	app := &App{
		Router:     router,
		Server:     srv,
		WsUpgrader: wsUpg,
		Hub:        hub,
		Log:        log,
		Storage:    db,
	}

	return app
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

	a.setupRoutes()

	if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

func (a *App) setupRoutes() {
	memberOnly := MWIsChatMember.New(a.Log, a.Storage)
	ownerOnly := MWIsChatOwner.New(a.Log, a.Storage)

	// Health check
	a.Router.Get("/health", health.New(context.Background(), a.Log))

	// WebSocket connection route
	a.Router.With(memberOnly).Get("/chats/{chatID}/ws", wsconnect.New(context.Background(), a.Log, a.WsUpgrader, a.Hub, a.Storage))

	// Chat routes
	a.Router.Get("/chats", getuserchat.New(context.Background(), a.Log, a.Storage))
	a.Router.Get("/chats-with-messages", getuserchatswithmessages.New(context.Background(), a.Log, a.Storage))
	a.Router.Post("/chats", createchat.New(context.Background(), a.Log, a.Storage))
	a.Router.Post("/chats/private", createprivatechat.New(context.Background(), a.Log, a.Storage))
	a.Router.With(ownerOnly).Put("/chats/{chatID}", updatechat.New(context.Background(), a.Log, a.Storage))
	a.Router.With(ownerOnly).Delete("/chats/{chatID}", deletechat.New(context.Background(), a.Log, a.Storage))
	a.Router.With(memberOnly).Get("/chats/{chatID}/members", getchatmembers.New(context.Background(), a.Log, a.Storage))
	a.Router.With(ownerOnly).Post("/chats/{chatID}/members", addmembers.New(context.Background(), a.Log, a.Storage))
	a.Router.With(ownerOnly).Delete("/chats/{chatID}/members", removemembers.New(context.Background(), a.Log, a.Storage))
	a.Router.Delete("/chats/{chatID}/leave", leavechat.New(context.Background(), a.Log, a.Storage))
	a.Router.Get("/chats/{chatID}/user-role", getuserrole.New(context.Background(), a.Log, a.Storage))
	a.Router.With(memberOnly).Get("/chats/{chatID}/messages", getmessages.New(context.Background(), a.Log, a.Storage))

	// Message routes
	a.Router.Post("/messages", createmessage.New(context.Background(), a.Log, a.Storage))
	a.Router.Put("/messages/{messageID}", updatemessage.New(context.Background(), a.Log, a.Storage))
	a.Router.Delete("/messages/{messageID}", deletemessage.New(context.Background(), a.Log, a.Storage))
	a.Router.Post("/messages/{messageID}/mark-as-read", markasread.New(context.Background(), a.Log, a.Storage))
	a.Router.Get("/messages/{messageID}/status", getmessagestatus.New(context.Background(), a.Log, a.Storage))

	// User routes
	a.Router.Post("/users", createuser.New(context.Background(), a.Log, a.Storage))
	a.Router.Get("/users/{userID}", getuser.New(context.Background(), a.Log, a.Storage))
	a.Router.Get("/users/by-name", getuserbyname.New(context.Background(), a.Log, a.Storage))
	a.Router.Put("/users/{userID}", updateuser.New(context.Background(), a.Log, a.Storage))
	a.Router.Delete("/users/{userID}", deleteuser.New(context.Background(), a.Log, a.Storage))
}

func setupMiddlewares(router *chi.Mux, log *slog.Logger) {
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(MWLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	//...add more middlewares here if needed
}
