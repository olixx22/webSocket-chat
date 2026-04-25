package getmessagestatus

import (
	"context"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/domain/models"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type StatusGetter interface {
	GetMessageStatus(ctx context.Context, messageID uuid.UUID) ([]models.MessageStatus, error)
}

type Response struct {
	resp.Response
	Statuses []models.MessageStatus `json:"statuses"`
}

func New(ctx context.Context, log *slog.Logger, statusGetter StatusGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Messages.GetMessageStatus.New"

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

		statuses, err := statusGetter.GetMessageStatus(ctx, messageID)
		if err != nil {
			log.Error("failed to get message status", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to get message status"))
			return
		}

		log.Info("message statuses retrieved successfully", slog.String("message_id", messageID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Statuses: statuses,
		})
	}
}
