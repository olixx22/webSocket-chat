package getuserchat

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

type ChatGetter interface {
	GetUserChats(context.Context, uuid.UUID) ([]models.Chat, error)
}

type Request struct {
	UserID string `validate:"required"`
}

type Response struct {
	resp.Response
	Chats []models.Chat `json:"chats"`
}

func New(ctx context.Context, log *slog.Logger, chatGetter ChatGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.GetUserChats.New"

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

		log.Info("request parsed", slog.Any("request", req))

		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid user ID"))
			return
		}

		chats, err := chatGetter.GetUserChats(ctx, userID)
		if err != nil {
			log.Error("failed to get user chats", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to retrieve chats"))
			return
		}

		log.Info("user chats retrieved", slog.Int("count", len(chats)))


//Debug
		log.Info("response debug", slog.Any("response", Response{
    Response: resp.OK(),
    Chats: chats,
}))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Chats:    chats,
		})
	}
}
