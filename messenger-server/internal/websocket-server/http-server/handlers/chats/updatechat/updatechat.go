package updatechat

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

type ChatUpdater interface {
	UpdateChatTitle(ctx context.Context, chatID uuid.UUID, title string) (models.Chat, error)
}

type Request struct {
	Title string `json:"title" validate:"required,min=1"`
}

type Response struct {
	resp.Response
	Chat models.Chat `json:"chat"`
}

func New(ctx context.Context, log *slog.Logger, chatUpdater ChatUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.UpdateChat.New"

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

		chat, err := chatUpdater.UpdateChatTitle(ctx, chatID, req.Title)
		if err != nil {
			if errors.Is(err, storage.ErrChatNotFound) {
				log.Error("chat not found", sl.Err(err))
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, resp.Error("Chat not found"))
				return
			}

			log.Error("failed to update chat", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to update chat"))
			return
		}

		log.Info("chat updated successfully", slog.String("chat_id", chat.ID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Chat:     chat,
		})
	}
}
