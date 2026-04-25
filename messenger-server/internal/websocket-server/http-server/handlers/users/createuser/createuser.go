package createuser

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/domain/models"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"
	"ws_chat/messenger-server/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
)

type UserCreator interface {
	CreateUser(context.Context, uuid.UUID, string, string, string) (models.User, error)
}

type Request struct {
	UserID    string `json:"user_id" validate:"required"`
	Username  string `json:"username" validate:"required,min=1"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
}

type Response struct {
	resp.Response
	User models.User `json:"user"`
}

func New(ctx context.Context, log *slog.Logger, userCreator UserCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Users.CreateUser.New"

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

		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			log.Error("failed to parse user ID", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		user, err := userCreator.CreateUser(ctx, userID, req.Username, req.AvatarURL, req.Bio)
		if err != nil {
			if errors.Is(err, storage.ErrUserExists) {
				log.Error("user already exists", sl.Err(err))
				w.WriteHeader(http.StatusConflict)
				render.JSON(w, r, resp.Error("User already exists"))
				return
			}

			log.Error("failed to create user", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to create user"))
			return
		}

		log.Info("user created successfully", slog.String("user_id", user.ID.String()))

		w.WriteHeader(http.StatusCreated)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			User:     user,
		})
	}
}
