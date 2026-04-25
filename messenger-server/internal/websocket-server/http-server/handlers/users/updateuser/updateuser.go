package updateuser

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

type UserUpdater interface {
	UpdateUserByID(context.Context, models.User) (models.User, error)
}

type Request struct {
	Username  string `json:"username" validate:"required,min=1"`
	AvatarURL *string `json:"avatar_url"`
	Bio       *string `json:"bio"`
}

type Response struct {
	resp.Response
	User models.User `json:"user"`
}

func New(ctx context.Context, log *slog.Logger, userUpdater UserUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Users.UpdateUser.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID, err := uuid.Parse(chi.URLParam(r, "userID"))
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid user ID"))
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

		user := models.User{
			ID:        userID,
			Username:  req.Username,
			AvatarURL: req.AvatarURL,
			Bio:       req.Bio,
		}

		updatedUser, err := userUpdater.UpdateUserByID(ctx, user)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				log.Error("user not found", sl.Err(err))
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, resp.Error("User not found"))
				return
			}

			log.Error("failed to update user", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to update user"))
			return
		}

		log.Info("user updated successfully", slog.String("user_id", updatedUser.ID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			User:     updatedUser,
		})
	}
}
