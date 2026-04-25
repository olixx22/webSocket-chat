package getuserrole

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

type RoleGetter interface {
	GetUserRole(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) (models.MemberRole, error)
}

type Request struct {
	UserID string `json:"user_id" validate:"required"`
}

type Response struct {
	resp.Response
	Role models.MemberRole `json:"role"`
}

func New(ctx context.Context, log *slog.Logger, roleGetter RoleGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.GetUserRole.New"

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

		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			log.Error("invalid user ID", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid user ID"))
			return
		}

		role, err := roleGetter.GetUserRole(ctx, chatID, userID)
		if err != nil {
			if errors.Is(err, storage.ErrUserIsNotMember) {
				log.Error("user is not a member of the chat", sl.Err(err))
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, resp.Error("User is not a member of this chat"))
				return
			}

			log.Error("failed to get user role", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to get user role"))
			return
		}

		log.Info("user role retrieved successfully", slog.String("chat_id", chatID.String()), slog.String("role", string(role)))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Role:     role,
		})
	}
}
