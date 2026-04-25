package leavechat

import (
	"context"
	"log/slog"
	"net/http"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
)

type ChatLeaver interface {
	LeaveChat(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

type Request struct {
	UserID string `validate:"required"`
}

type Response struct {
	resp.Response
	Message string `json:"message"`
}

func New(ctx context.Context, log *slog.Logger, chatLeaver ChatLeaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.LeaveChat.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		req := Request{
			UserID: r.URL.Query().Get("user_id"),
		}

		if err := validator.New().Struct(req); err != nil {
			validationErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid request"))
			render.JSON(w, r, resp.ValidationError(validationErr))
			return
		}

		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid user ID"))
			return
		}

		chatID, err := uuid.Parse(chi.URLParam(r, "chatID"))
		if err != nil {
			log.Error("invalid chat id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid chat ID"))
			return
		}

		success, err := chatLeaver.LeaveChat(ctx, chatID, userID)
		if err != nil {
			log.Error("failed to leave chat", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to leave chat"))
			return
		}

		if !success {
			log.Error("user is not member of the chat")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("User is not a member of this chat"))
			return
		}

		log.Info("user left chat successfully", slog.String("chat_id", chatID.String()), slog.String("user_id", userID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Message:  "Successfully left the chat",
		})
	}
}
