package getuserchatswithmessages

import (
	"context"
	"log/slog"
	"net/http"
	"ws_chat/messenger-server/internal/domain/models"
	resp "ws_chat/messenger-server/internal/lib/api/response"
	"ws_chat/messenger-server/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type ChatListGetter interface {
	GetUserChatsWithLastMessage(ctx context.Context, userID uuid.UUID) ([]models.ChatPreview, error)
}

type Response struct {
	resp.Response
	Chats []models.ChatPreview `json:"chats"`
}

func New(ctx context.Context, log *slog.Logger, chatListGetter ChatListGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.GetUserChatsWithMessages.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		cookie, err := r.Cookie("userID")
		if err != nil {
			log.Error("cookie with user ID not found")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Cookie not found"))
			return
		}

		userID, err := uuid.Parse(cookie.Value)
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Invalid user ID"))
			return
		}

		chats, err := chatListGetter.GetUserChatsWithLastMessage(ctx, userID)
		if err != nil {
			log.Error("failed to get user chats with messages", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Failed to get chats"))
			return
		}

		log.Info("user chats with messages retrieved successfully", slog.String("user_id", userID.String()))

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Chats:    chats,
		})
	}
}
