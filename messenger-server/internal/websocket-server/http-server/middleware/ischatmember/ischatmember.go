package MWIsChatMember

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"
	"ws_chat/messenger-server/internal/storage"

	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type MemberChecker interface {
	IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

func New(log *slog.Logger, memberChecker MemberChecker) func(next http.Handler) http.Handler {
	log = log.With(
		slog.String("component", "middleware/is-chat-member"),
	)

	log.Info("is chat member middleware enabled")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatID, err := parseChatID(r)
			if err != nil {
				log.Error("invalid chat id", sl.Err(err))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid chat ID"))
				return
			}

			userID, err := parseUserID(r)
			if err != nil {
				log.Error("invalid user id", sl.Err(err))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error(err.Error()))
				return
			}

			isMember, err := memberChecker.IsMember(r.Context(), chatID, userID)
			if err != nil {
				log.Error("failed to check chat membership", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("Unknown error"))
				return
			}

			if !isMember {
				log.Error("user is not member of the chat")
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, resp.Error(storage.ErrUserIsNotMember.Error()))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseChatID(r *http.Request) (uuid.UUID, error) {
	chatIDStr := r.URL.Query().Get("chat_id")
	if chatIDStr == "" {
		return uuid.Nil, errors.New("chat id is required")
	}
	return uuid.Parse(chatIDStr)
}

func parseUserID(r *http.Request) (uuid.UUID, error) {
	if cookie, err := r.Cookie("userID"); err == nil {
		return uuid.Parse(cookie.Value)
	}

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		return uuid.Parse(userIDStr)
	}

	return uuid.Nil, errors.New("user id is required")
}
