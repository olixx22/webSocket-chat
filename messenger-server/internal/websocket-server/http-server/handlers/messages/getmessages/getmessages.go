package getmessages

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"ws_chat/messenger-server/internal/domain/models"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
)

type ChatGetter interface {
	GetMessages(context.Context, uuid.UUID, *string, int) ([]models.Message, error)
}

type Request struct {
	UserID string `validate:"required"`
}

type Response struct {
	resp.Response
	Messages []models.Message `json:"messages"`
}

func New(ctx context.Context, log *slog.Logger, chatGetter ChatGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Messages.GetMessages.New"

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

		chatID, err := uuid.Parse(chi.URLParam(r, "chatID"))
		if err != nil {
			log.Error("invalid chat id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid request"))
			return
		}

		limit := 50
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsedLimit, convErr := strconv.Atoi(rawLimit)
			if convErr != nil || parsedLimit <= 0 || parsedLimit > 200 {
				log.Error("invalid limit", sl.Err(errors.New("limit must be between 1 and 200")))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid request"))
				return
			}

			limit = parsedLimit
		}

		var before *string
		if rawBefore := r.URL.Query().Get("before"); rawBefore != "" {
			if _, parseErr := time.Parse(time.RFC3339, rawBefore); parseErr != nil {
				log.Error("invalid before timestamp", sl.Err(parseErr))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid request"))
				return
			}

			before = &rawBefore
		}

		log.Info("getting chat messages...")

		messages, err := chatGetter.GetMessages(ctx, chatID, before, limit)
		if err != nil {
			log.Error("failed to get messages", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		log.Info("messages received", slog.Int("count", len(messages)))
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Messages: messages,
		})
	}
}
