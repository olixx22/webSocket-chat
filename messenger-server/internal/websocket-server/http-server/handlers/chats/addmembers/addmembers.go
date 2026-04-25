package addmembers

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

type MembersAdder interface {
	AddMembersWithRoles(ctx context.Context, chatID uuid.UUID, members []models.MemberInput) (success bool, err error)
}

type Request struct {
	Members []models.MemberInput `json:"members" validate:"required,min=1"`
}

type Response struct {
	resp.Response
	Success bool `json:"success"`
}

func New(ctx context.Context, log *slog.Logger, membersAdder MembersAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Chats.AddMembers.New"

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

		success, err := membersAdder.AddMembersWithRoles(ctx, chatID, req.Members)
		if err != nil {
			if errors.Is(err, storage.ErrEmptyMemberInput) {
				log.Error("empty member input")
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Empty member input"))
				return 
			}

			if errors.Is(err, storage.ErrInvalidRole) {
				log.Error("invalid role", sl.Err(err))
				w.WriteHeader(http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Invalid role for one of the members"))
				return 
			}

			if errors.Is(err, storage.ErrChatOrUserNotFound) {
				log.Error("chat or user not found", sl.Err(err))
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, resp.Error("Chat or one of the users not found"))
				return
			}

			if errors.Is(err, storage.ErrDuplicateMember) {
				log.Error("duplicate member", sl.Err(err))
				w.WriteHeader(http.StatusUnprocessableEntity)
				render.JSON(w, r, resp.Error("One of the users is already in chat"))
				return
			}

			log.Error("internal error", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("Unknown error"))
			return
		}

		if success {
			log.Info("users were added to the chat successfully")
			w.WriteHeader(http.StatusOK)
			render.JSON(w, r, Response{
				Response: resp.OK(),
				Success: success,
			})
		} else {
			log.Warn("users were not added to the chat!")
			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, Response{
				Response: resp.Error("Users were not added"),
				Success: success,
			})
		}
	}
}