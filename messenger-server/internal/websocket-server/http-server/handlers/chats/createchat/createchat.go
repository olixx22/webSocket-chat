package createchat

import (
	"context"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/domain/models"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
)

type ChatCreator interface {
	CreateChat(ctx context.Context, creatorID uuid.UUID, chatID uuid.UUID, title string, isGroup bool) (chat models.Chat, err error)
}

type Request struct {
	Title     string `json:"title" validate:"required,min=1"`
	IsGroup   bool   `json:"is_group"`
}

type Response struct {
	resp.Response
	Chat models.Chat `json:"chat"`
}

func New(ctx context.Context, log *slog.Logger, chatCreator ChatCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.CreateChat.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

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

		cookie, err := r.Cookie("userID")
		if err != nil {
			log.Error("cookie with user ID not found")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Cookie not found"))
			return
		}

		creatorId := cookie.Value

		creatorID, err := uuid.Parse(creatorId)
		if err != nil {
			log.Error("invalid creator id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid creator ID"))
			return
		}

		chatID := uuid.New()

		chat, err := chatCreator.CreateChat(ctx, creatorID, chatID, req.Title, req.IsGroup)
		if err != nil {
			log.Error("failed to create chat", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to create chat"))
			return
		}

		log.Info("chat created successfully", slog.String("chat_id", chat.ID.String()))

		w.WriteHeader(http.StatusCreated)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Chat:     chat,
		})
	}
}
