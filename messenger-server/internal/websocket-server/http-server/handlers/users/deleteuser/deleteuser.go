package deleteuser

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
	"github.com/google/uuid"
)

type UserDeleter interface {
	DeleteUserByID(context.Context, uuid.UUID) (bool, error)
}

type Response struct {
	resp.Response
	Message string `json:"message"`
}

func New(ctx context.Context, log *slog.Logger, userDeleter UserDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Users.DeleteUser.New"

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

		success, err := userDeleter.DeleteUserByID(ctx, userID)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				log.Error("user not found", sl.Err(err))
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, resp.Error("User not found"))
				return
			}

			log.Error("failed to delete user", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to delete user"))
			return
		}

		if !success {
			log.Error("user not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, resp.Error("User not found"))
			return
		}

		log.Info("user deleted successfully", slog.String("user_id", userID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Message:  "User deleted successfully",
		})
	}
}
