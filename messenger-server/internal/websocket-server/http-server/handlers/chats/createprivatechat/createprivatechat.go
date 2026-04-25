package createprivatechat

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

type PrivateChatCreator interface {
	CreatePrivateChat(ctx context.Context, user1, user2 uuid.UUID, title string) (models.Chat, error)
}

type Request struct {
	UserID string `json:"user_id" validate:"required"`
	Title string `json:"title" validate:"required"`
}

type Response struct {
	resp.Response
	Chat models.Chat `json:"chat"`
}

func New(ctx context.Context, log *slog.Logger, privateChatCreator PrivateChatCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.CreatePrivateChat.New"

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

		user1ID, err := uuid.Parse(cookie.Value)
		if err != nil {
			log.Error("invalid user id from cookie", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid user ID"))
			return
		}

		user2ID, err := uuid.Parse(req.UserID)
		if err != nil {
			log.Error("invalid user id from request", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid user ID"))
			return
		}

		chat, err := privateChatCreator.CreatePrivateChat(ctx, user1ID, user2ID, req.Title)
		if err != nil {
			log.Error("failed to create private chat", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to create private chat"))
			return
		}

		log.Info("private chat created successfully", slog.String("chat_id", chat.ID.String()))

		w.WriteHeader(http.StatusCreated)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Chat:     chat,
		})
	}
}
