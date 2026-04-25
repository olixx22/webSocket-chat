package getchatmembers

import (
	"context"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/domain/models"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
)

type MembersGetter interface {
	GetMembers(ctx context.Context, chatID uuid.UUID) (members []models.User, err error)
}

type Request struct {
	UserID string `validate:"required"`
}

type Response struct {
	resp.Response
	Members []models.User `json:"members"`
}

func New(ctx context.Context, log *slog.Logger, membersGetter MembersGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.GetChatMembers.New"

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

		chatID, err := uuid.Parse(chi.URLParam(r, "chatID"))
		if err != nil {
			log.Error("invalid chat id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid chat ID"))
			return
		}

		members, err := membersGetter.GetMembers(ctx, chatID)
		if err != nil {
			log.Error("failed to get chat members", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to retrieve members"))
			return
		}

		if members == nil {
			members = make([]models.User, 0)
		}

		log.Info("chat members retrieved", slog.Int("count", len(members)))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Members:  members,
		})
	}
}
