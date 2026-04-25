package deletemessage

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

type MessageDeleter interface {
	DeleteMessage(ctx context.Context, messageID uuid.UUID) (bool, error)
}

type Response struct {
	resp.Response
	Success bool `json:"success"`
}

func New(ctx context.Context, log *slog.Logger, messageDeleter MessageDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Messages.DeleteMessage.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		messageIDStr := chi.URLParam(r, "messageID")
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			log.Error("invalid message ID", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid message ID"))
			return
		}

		success, err := messageDeleter.DeleteMessage(ctx, messageID)
		if err != nil {
			if errors.Is(err, storage.ErrMessageNotFound) {
				log.Error("message not found", sl.Err(err))
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, resp.Error("Message not found"))
				return
			}

			log.Error("failed to delete message", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to delete message"))
			return
		}

		log.Info("message deleted successfully", slog.String("message_id", messageID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Success:  success,
		})
	}
}
