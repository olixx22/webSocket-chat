package deletechat

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"
	"ws_chat/messenger-server/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type ChatDeleter interface {
	DeleteChatByID(ctx context.Context, chatID uuid.UUID) (bool, error)
}

type Response struct {
	resp.Response
	Success bool `json:"success"`
}

func New(ctx context.Context, log *slog.Logger, chatDeleter ChatDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.DeleteChat.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		chatIDStr := chi.URLParam(r, "chatID")
		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			log.Error("invalid chat ID", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid chat ID"))
			return
		}

		success, err := chatDeleter.DeleteChatByID(ctx, chatID)
		if err != nil {
			if errors.Is(err, storage.ErrChatNotFound) {
				log.Error("chat not found", sl.Err(err))
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, resp.Error("Chat not found"))
				return
			}

			log.Error("failed to delete chat", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to delete chat"))
			return
		}

		log.Info("chat deleted successfully", slog.String("chat_id", chatID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Success:  success,
		})
	}
}
