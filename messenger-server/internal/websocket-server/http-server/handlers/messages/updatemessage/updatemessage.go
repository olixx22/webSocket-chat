package updatemessage

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/domain/models"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"
	"ws_chat/messenger-server/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
)

type MessageUpdater interface {
	UpdateMessage(ctx context.Context, messageID uuid.UUID, content string) (models.Message, error)
}

type Request struct {
	Content string `json:"content" validate:"required,min=1"`
}

type Response struct {
	resp.Response
	Message models.Message `json:"message"`
}

func New(ctx context.Context, log *slog.Logger, messageUpdater MessageUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Messages.UpdateMessage.New"

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

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid request body"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			validationErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid request"))
			render.JSON(w, r, resp.ValidationError(validationErr))
			return
		}

		msg, err := messageUpdater.UpdateMessage(ctx, messageID, req.Content)
		if err != nil {
			if errors.Is(err, storage.ErrMessageNotFound) {
				log.Error("message not found", sl.Err(err))
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, resp.Error("Message not found"))
				return
			}

			log.Error("failed to update message", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to update message"))
			return
		}

		log.Info("message updated successfully", slog.String("message_id", msg.ID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Message:  msg,
		})
	}
}
