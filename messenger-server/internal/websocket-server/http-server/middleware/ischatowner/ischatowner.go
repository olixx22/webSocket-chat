package MWIsChatOwner

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
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type OwnerChecker interface {
	GetUserRole(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) (models.MemberRole, error)
}

func New(log *slog.Logger, ownerChecker OwnerChecker) func(next http.Handler) http.Handler {
	log = log.With(
		slog.String("component", "middleware/is-chat-owner"),
	)

	log.Info("is chat owner middleware enabled")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatIDStr := chi.URLParam(r, "chatID")
			chatID, err := uuid.Parse(chatIDStr)
			if err != nil {
				log.Error("invalid chat id", sl.Err(err))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid chat ID"))
				return
			}

			cookie, err := r.Cookie("userID")
			if err != nil {
				log.Error("cookie with user id not found", sl.Err(err))
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, resp.Error("Cookie not found"))
				return
			}

			userID, err := uuid.Parse(cookie.Value)
			if err != nil {
				log.Error("invalid user id in cookie", sl.Err(err))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid user ID"))
				return
			}

			role, err := ownerChecker.GetUserRole(r.Context(), chatID, userID)
			if err != nil {
				if errors.Is(err, storage.ErrUserIsNotMember) {
					log.Error("user is not a member of the chat", sl.Err(err))
					w.WriteHeader(http.StatusForbidden)
					render.JSON(w, r, resp.Error("User is not a member of this chat"))
					return
				}

				log.Error("failed to get user role", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("Unknown error"))
				return
			}

			if role != models.RoleOwner {
				log.Error("user is not owner of the chat", slog.String("role", string(role)))
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, resp.Error("User is not owner of this chat"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
