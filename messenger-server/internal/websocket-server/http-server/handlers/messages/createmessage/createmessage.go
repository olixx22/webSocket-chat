package createmessage

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

type MessageCreator interface {
	CreateMessage(ctx context.Context, msg models.Message) (models.Message, error)
}

type Request struct {
	Content     string `json:"content" validate:"required,min=1"`
	MessageType string `json:"message_type" validate:"required"`
}

type Response struct {
	resp.Response
	Message models.Message `json:"message"`
}

func New(ctx context.Context, log *slog.Logger, messageCreator MessageCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Messages.CreateMessage.New"

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

		cookie, err := r.Cookie("userID")
		if err != nil {
			log.Error("cookie with user ID not found")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Cookie not found"))
			return
		}

		senderID, err := uuid.Parse(cookie.Value)
		if err != nil {
			log.Error("invalid sender id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid sender ID"))
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

		msg := models.Message{
			ID:        uuid.New(),
			ChatID:    chatID,
			SenderID:  &senderID,
			Content:   &req.Content,
			Type:      req.MessageType,
		}

		created, err := messageCreator.CreateMessage(ctx, msg)
		if err != nil {
			if errors.Is(err, storage.ErrUserIsNotMember) {
				log.Error("user is not a member of the chat", sl.Err(err))
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, resp.Error("User is not a member of this chat"))
				return
			}

			log.Error("failed to create message", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to create message"))
			return
		}

		log.Info("message created successfully", slog.String("message_id", created.ID.String()))

		w.WriteHeader(http.StatusCreated)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Message:  created,
		})
	}
}
