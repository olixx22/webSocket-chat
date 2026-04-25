package removemembers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"
	"ws_chat/messenger-server/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
)

type MembersRemover interface {
	RemoveMembers(ctx context.Context, chatID uuid.UUID, userIDs []uuid.UUID) (bool, error)
}

type Request struct {
	UserIDs []string `json:"user_ids" validate:"required,min=1"`
}

type Response struct {
	resp.Response
	Success bool `json:"success"`
}

func New(ctx context.Context, log *slog.Logger, membersRemover MembersRemover) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.RemoveMembers.New"

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

		userIDs := make([]uuid.UUID, len(req.UserIDs))
		for i, userIDStr := range req.UserIDs {
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				log.Error("invalid user ID", sl.Err(err))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid user ID format"))
				return
			}
			userIDs[i] = userID
		}

		success, err := membersRemover.RemoveMembers(ctx, chatID, userIDs)
		if err != nil {
			if errors.Is(err, storage.ErrEmptyUserIDs) {
				log.Error("empty user IDs", sl.Err(err))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("User IDs cannot be empty"))
				return
			}
			if errors.Is(err, storage.ErrNoMembersRemoved) {
				log.Error("no members removed", sl.Err(err))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("No members were removed"))
				return
			}

			log.Error("failed to remove members", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to remove members"))
			return
		}

		log.Info("members removed successfully", slog.String("chat_id", chatID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Success:  success,
		})
	}
}
