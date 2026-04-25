package connect

import (
	"context"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/domain/models"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"
	"ws_chat/messenger-server/internal/websocket-server/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Connector interface {
	CreateMessage(context.Context, models.Message) (models.Message, error)
}

type Request struct {}

func New(ctx context.Context, log *slog.Logger, upgrader *websocket.Upgrader, hub *ws.Hub, connector Connector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.WS.Connect.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		if err := validator.New().Struct(req); err != nil {
			validationErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid request"))
			render.JSON(w, r, resp.ValidationError(validationErr))
			return
		}

		log.Info("request parsed", slog.Any("request", req))

		cookie, err := r.Cookie("userID")
		if err != nil {
			log.Error("cookie not found", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Cookie not found"))
			return
		}

		userId := cookie.Value

		userID, err := uuid.Parse(userId)
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid request"))
			return
		}

		chatId := chi.URLParam(r, "chatID")

		chatID, err := uuid.Parse(chatId)
		if err != nil {
			log.Error("invalid chat id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid request"))
			return
		}

		log.Info("upgrading websocket connection...")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("failed to upgrade websocket connection", sl.Err(err))
			return
		}

		client := ws.NewClient(log, hub, connector, conn, userID, chatID)
		hub.Register(client)

		go client.WritePump()
		go client.ReadPump()
	}
}
