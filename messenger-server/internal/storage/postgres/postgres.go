package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"ws_chat/messenger-server/internal/domain/models"
	"ws_chat/messenger-server/internal/lib/helpers"
	"ws_chat/messenger-server/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func New(ctx context.Context, dbURL string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close(ctx context.Context) error {
	const op = "storage.postgres.Close"

	s.db.Close()

	return nil
}

const (
	NotFoundCode      = "23503"
	AlreadyExistsCode = "23505"
)

func (s *Storage) CreateUser(ctx context.Context, id uuid.UUID, username string, avatarURL string, bio string) (user models.User, err error) {
	const op = "storage.postgres.CreateUser"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	_, err = tx.Exec(ctx, "INSERT INTO users(id, username, avatar_url, bio) VALUES ($1, $2, $3, $4)", id, username, avatarURL, bio)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	user = models.User{
		ID:        id,
		Username:  username,
		AvatarURL: &avatarURL,
		Bio:       &bio,
	}

	return user, nil
}

func (s *Storage) UserByID(ctx context.Context, id uuid.UUID) (user models.User, err error) {
	const op = "storage.postgres.UserByID"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	err = tx.QueryRow(ctx, "SELECT id, username, avatar_url, bio FROM users WHERE id = $1", id).Scan(&user.ID, &user.Username, &user.AvatarURL, &user.Bio)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) UserByName(ctx context.Context, username string) (user models.User, err error) {
	const op = "storage.postgres.UserByName"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	err = tx.QueryRow(ctx, "SELECT id, username, avatar_url, bio FROM users WHERE username = $1", username).Scan(&user.ID, &user.Username, &user.AvatarURL, &user.Bio)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) UpdateUserByID(ctx context.Context, user models.User) (updUser models.User, err error) {
	const op = "storage.postgres.UpdateUser"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	err = tx.QueryRow(ctx, "UPDATE users SET username = COALESCE($1, username), avatar_url = COALESCE($2, avatar_url), bio = COALESCE($3, bio) WHERE id = $4 RETURNING id, username, avatar_url, bio", user.Username, user.AvatarURL, user.Bio, user.ID).Scan(&updUser.ID, &updUser.Username, &updUser.AvatarURL, &updUser.Bio)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return updUser, nil
}

func (s *Storage) DeleteUserByID(ctx context.Context, id uuid.UUID) (success bool, err error) {
	const op = "storage.postgres.DeleteUserByID"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM users WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	if !exists {
		return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
	}

	_, err = tx.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (s *Storage) CreateChat(ctx context.Context, creatorID uuid.UUID, chatID uuid.UUID, title string, isGroup bool) (chat models.Chat, err error) {
	const op = "storage.postgres.CreateChat"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	_, err = tx.Exec(ctx, "INSERT INTO chats(id, is_group, title) VALUES ($1, $2, $3)", chatID, isGroup, title)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == AlreadyExistsCode {
			return models.Chat{}, fmt.Errorf("%s: %w", op, storage.ErrChatExists)
		}
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO chat_members (chat_id, user_id, role) VALUES ($1, $2, $3)", chatID, creatorID, models.RoleOwner)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == AlreadyExistsCode {
			return models.Chat{}, fmt.Errorf("%s: %w", op, storage.ErrUserIsAlreadyInChat)
		}
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	creator, err := s.UserByID(ctx, creatorID)
	if err != nil {
		return models.Chat{}, err
	}

	chat = models.Chat{
		ID:      chatID,
		IsGroup: isGroup,
		Title:   title,
		Members: []models.User{creator},
	}

	return chat, nil
}

func (s *Storage) CreatePrivateChat(ctx context.Context, user1, user2 uuid.UUID, title string) (chat models.Chat, err error) {
	const op = "storage.postgres.CreatePrivateChat"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	var existingChatID uuid.UUID

	err = tx.QueryRow(ctx, `
		SELECT cm1.chat_id
		FROM chat_members cm1
		JOIN chat_members cm2 ON cm1.chat_id = cm2.chat_id
		JOIN chats c ON c.id = cm1.chat_id
		WHERE c.is_group = false
		  AND (
		        (cm1.user_id = $1 AND cm2.user_id = $2)
		     OR (cm1.user_id = $2 AND cm2.user_id = $1)
		  )
		LIMIT 1
	`, user1, user2).Scan(&existingChatID)

	if err == nil {
		chat, err = getChatFull(ctx, tx, existingChatID)
		if err != nil {
			return models.Chat{}, fmt.Errorf("%s: %w", op, err)
		}
		return chat, nil
	}

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	chatID := uuid.New()

	_, err = tx.Exec(ctx, `
		INSERT INTO chats (id, is_group, title)
		VALUES ($1, false, $2)
	`, chatID, title)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO chat_members (chat_id, user_id)
		VALUES ($1, $2), ($1, $3)
	`, chatID, user1, user2)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	chat, err = getChatFull(ctx, tx, chatID)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	return chat, nil
}

func (s *Storage) ChatByID(ctx context.Context, chatID uuid.UUID) (models.Chat, error) {
	const op = "storage.postgres.ChatByID"

	chat, err := getChatFull(ctx, s.db, chatID)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	return chat, nil
}

func getChatFull(ctx context.Context, q Querier, chatID uuid.UUID) (chat models.Chat, err error) {
	const op = "storage.postgres.getChatFull"

	err = q.QueryRow(ctx,
		"SELECT id, is_group, title FROM chats WHERE id = $1",
		chatID,
	).Scan(&chat.ID, &chat.IsGroup, &chat.Title)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Chat{}, fmt.Errorf("%s: %w", op, storage.ErrChatNotFound)
		}
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	chat.Members, err = getMembers(ctx, q, chatID)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	return chat, nil
}

func (s *Storage) GetMembers(ctx context.Context, chatID uuid.UUID) ([]models.User, error) {
	const op = "storage.postgres.GetMembers"

	members, err := getMembers(ctx, s.db, chatID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return members, nil
}

func getMembers(ctx context.Context, q Querier, chatID uuid.UUID) (members []models.User, err error) {
	const op = "storage.postgres.getMembers"

	rows, err := q.Query(ctx, "SELECT u.id, u.username, u.avatar_url, u.bio FROM users u JOIN chat_members cm ON u.id = cm.user_id WHERE cm.chat_id = $1", chatID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var u models.User

		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.AvatarURL,
			&u.Bio,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		members = append(members, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return members, nil
}

func (s *Storage) UpdateChatTitle(ctx context.Context, chatID uuid.UUID, title string) (chat models.Chat, err error) {
	const op = "storage.postgres.UpdateChatTitle"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	err = tx.QueryRow(ctx, "UPDATE chats SET title = COALESCE($1, title) WHERE id = $2 RETURNING id, is_group, title", title, chatID).Scan(&chat.ID, &chat.IsGroup, &chat.Title)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Chat{}, fmt.Errorf("%s: %w", op, storage.ErrChatNotFound)
		}
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	members, err := s.GetMembers(ctx, chatID)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	chat.Members = members

	return chat, nil
}

func (s *Storage) DeleteChatByID(ctx context.Context, chatID uuid.UUID) (success bool, err error) {
	const op = "storage.postgres.DeleteChatByID"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM chats WHERE id = $1)", chatID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	if !exists {
		return false, fmt.Errorf("%s: %w", op, storage.ErrChatNotFound)
	}

	_, err = tx.Exec(ctx, "DELETE FROM chats WHERE id = $1", chatID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (s *Storage) AddMembersWithRoles(ctx context.Context, chatID uuid.UUID, members []models.MemberInput) (success bool, err error) {
	const op = "storage.postgres.AddMembersWithRoles"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	if len(members) == 0 {
		return false, fmt.Errorf("%s: %w", op, storage.ErrEmptyMemberInput)
	}

	userIDs := make([]uuid.UUID, 0, len(members))
	roles := make([]models.MemberRole, 0, len(members))

	for _, m := range members {
		switch m.Role {
		case "owner", "admin", "member":
		default:
			return false, fmt.Errorf("%s: %w %q", op, storage.ErrInvalidRole, m.Role)
		}

		userIDs = append(userIDs, m.UserID)
		roles = append(roles, m.Role)
	}

	_, err = tx.Exec(ctx, "INSERT INTO chat_members (chat_id, user_id, role) SELECT $1, u.user_id, u.role FROM UNNEST($2::uuid[], $3::text[]) AS u(user_id, role) ON CONFLICT (chat_id, user_id) DO UPDATE SET role = EXCLUDED.role", chatID, userIDs, roles)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case NotFoundCode:
				return false, fmt.Errorf("%s: %w", op, storage.ErrChatOrUserNotFound)
			case AlreadyExistsCode:
				return false, fmt.Errorf("%s: %w", op, storage.ErrDuplicateMember)
			}
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (s *Storage) RemoveMembers(ctx context.Context, chatID uuid.UUID, userIDs []uuid.UUID) (success bool, err error) {
	const op = "storage.postgres.RemoveMembers"

	if len(userIDs) == 0 {
		return false, fmt.Errorf("%s: %w", op, storage.ErrEmptyUserIDs)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	stmt, err := tx.Exec(ctx, "DELETE FROM chat_members WHERE chat_id = $1 AND user_id = ANY($2::uuid[])", chatID, userIDs)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if stmt.RowsAffected() == 0 {
		return false, fmt.Errorf("%s: %w", op, storage.ErrNoMembersRemoved)
	}

	var count int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM chat_members WHERE chat_id = $1", chatID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if count == 0 {
		_, err = tx.Exec(ctx, "DELETE FROM chats WHERE id = $1", chatID)
		if err != nil {
			return false, fmt.Errorf("%s: %w", op, err)
		}
	}

	return true, nil
}

func (s *Storage) IsMember(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) (isMember bool, err error) {
	const op = "storage.postgres.IsMember"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	err = tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM chat_members WHERE chat_id = $1 AND user_id = $2)", chatID, userID).Scan(&isMember)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isMember, nil
}

func (s *Storage) GetUserRole(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) (role models.MemberRole, err error) {
	const op = "storage.postgres.GetUserRole"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	err = tx.QueryRow(ctx, "SELECT role FROM chat_members WHERE chat_id = $1 AND user_id = $2", chatID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, storage.ErrUserIsNotMember)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return role, nil
}

func (s *Storage) LeaveChat(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) (success bool, err error) {
	const op = "storage.postgres.LeaveChat"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	stmt, err := tx.Exec(ctx, "DELETE FROM chat_members WHERE chat_id = $1 AND user_id = $2", chatID, userID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if stmt.RowsAffected() == 0 {
		return false, fmt.Errorf("%s: %w", op, storage.ErrUserIsNotMember)
	}

	var count int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM chat_members WHERE chat_id = $1", chatID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if count == 0 {
		_, err = tx.Exec(ctx, "DELETE FROM chats WHERE id = $1", chatID)
		if err != nil {
			return false, fmt.Errorf("%s: %w", op, err)
		}
	}

	return true, nil
}

// Get all user's chats by user ID, important for the main page
func (s *Storage) GetUserChats(ctx context.Context, userID uuid.UUID) ([]models.Chat, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT c.id, c.is_group, c.title
		FROM chats c
		JOIN chat_members cm ON cm.chat_id = c.id
		WHERE cm.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.Chat
	var chatIDs []uuid.UUID

	for rows.Next() {
		var chat models.Chat

		if err = rows.Scan(&chat.ID, &chat.IsGroup, &chat.Title); err != nil {
			return nil, err
		}

		chats = append(chats, chat)
		chatIDs = append(chatIDs, chat.ID)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	for i := range chats {
		members, err := getMembers(ctx, tx, chatIDs[i])
		if err != nil {
			return nil, err
		}
		chats[i].Members = members
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return chats, nil
}

func (s *Storage) CreateMessage(ctx context.Context, msg models.Message) (created models.Message, err error) {
	const op = "storage.postgres.CreateMessage"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.Message{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	var isMember bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM chat_members 
			WHERE chat_id = $1 AND user_id = $2
		)
	`, msg.ChatID, msg.SenderID).Scan(&isMember)

	if err != nil {
		return models.Message{}, fmt.Errorf("%s: %w", op, err)
	}

	if !isMember {
		return models.Message{}, fmt.Errorf("%s: %w", op, storage.ErrUserIsNotMember)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO messages (id, chat_id, sender_id, content, message_type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, chat_id, sender_id, content, message_type, created_at
	`, msg.ID, msg.ChatID, msg.SenderID, msg.Content, msg.Type).Scan(
		&created.ID,
		&created.ChatID,
		&created.SenderID,
		&created.Content,
		&created.Type,
		&created.CreatedAt,
	)
	if err != nil {
		return models.Message{}, fmt.Errorf("%s: %w", op, err)
	}

	return created, nil
}

func (s *Storage) GetMessages(ctx context.Context, chatID uuid.UUID, lastTime *string, limit int) (messages []models.Message, err error) {
	const op = "storage.postgres.GetMessages"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	var rows pgx.Rows

	if lastTime == nil {
		rows, err = tx.Query(ctx, `
			SELECT id, chat_id, sender_id, content, message_type, created_at
			FROM messages
			WHERE chat_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`, chatID, limit)
	} else {
		rows, err = tx.Query(ctx, `
			SELECT id, chat_id, sender_id, content, message_type, created_at
			FROM messages
			WHERE chat_id = $1 AND created_at < $2
			ORDER BY created_at DESC
			LIMIT $3
		`, chatID, *lastTime, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var m models.Message

		err := rows.Scan(
			&m.ID,
			&m.ChatID,
			&m.SenderID,
			&m.Content,
			&m.Type,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return messages, nil
}

func (s *Storage) UpdateMessage(ctx context.Context, messageID uuid.UUID, content string) (msg models.Message, err error) {
	const op = "storage.postgres.UpdateMessage"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.Message{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	err = tx.QueryRow(ctx, `
		UPDATE messages
		SET content = $1
		WHERE id = $2
		RETURNING id, chat_id, sender_id, content, message_type, created_at
	`, content, messageID).Scan(
		&msg.ID,
		&msg.ChatID,
		&msg.SenderID,
		&msg.Content,
		&msg.Type,
		&msg.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Message{}, fmt.Errorf("%s: %w", op, storage.ErrMessageNotFound)
		}
		return models.Message{}, fmt.Errorf("%s: %w", op, err)
	}

	return msg, nil
}

func (s *Storage) DeleteMessage(ctx context.Context, messageID uuid.UUID) (success bool, err error) {
	const op = "storage.postgres.DeleteMessage"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	cmd, err := tx.Exec(ctx, `
		DELETE FROM messages WHERE id = $1
	`, messageID)

	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if cmd.RowsAffected() == 0 {
		return false, fmt.Errorf("%s: %w", op, storage.ErrMessageNotFound)
	}

	return true, nil
}

func (s *Storage) MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) (err error) {
	const op = "storage.postgres.MarkAsRead"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}

		commitErr := tx.Commit(ctx)
		if commitErr != nil {
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	_, err = tx.Exec(ctx, `
		INSERT INTO message_status (message_id, user_id, status)
		VALUES ($1, $2, 'read')
		ON CONFLICT (message_id, user_id)
		DO UPDATE SET status = 'read'
	`, messageID, userID)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetMessageStatus(ctx context.Context, messageID uuid.UUID) ([]models.MessageStatus, error) {
	const op = "storage.postgres.GetMessageStatus"

	rows, err := s.db.Query(ctx, `
		SELECT message_id, user_id, status
		FROM message_status
		WHERE message_id = $1
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []models.MessageStatus

	for rows.Next() {
		var ms models.MessageStatus
		err := rows.Scan(&ms.MessageID, &ms.UserID, &ms.Status)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, ms)
	}

	return result, nil
}

func (s *Storage) GetUserChatsWithLastMessage(ctx context.Context, userID uuid.UUID) ([]models.ChatPreview, error) {
	const op = "storage.postgres.GetUserChatsWithLastMessage"

	rows, err := s.db.Query(ctx, `
		SELECT 
			c.id, c.is_group, c.title,
			m.id, m.chat_id, m.sender_id, m.content, m.message_type, m.created_at
		FROM chats c
		JOIN chat_members cm ON cm.chat_id = c.id
		LEFT JOIN LATERAL (
			SELECT id, chat_id, sender_id, content, message_type, created_at
			FROM messages
			WHERE chat_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON true
		WHERE cm.user_id = $1
	`, userID)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var chats []models.ChatPreview

	for rows.Next() {
		var cp models.ChatPreview
		var msgID *uuid.UUID
		var msgChatID *uuid.UUID
		var senderID *uuid.UUID
		var content *string
		var messageType *string
		var createdAt *time.Time

		err := rows.Scan(
			&cp.ID,
			&cp.IsGroup,
			&cp.Title,
			&msgID,
			&msgChatID,
			&senderID,
			&content,
			&messageType,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		if msgID != nil {
			cp.LastMessage = &models.Message{
				ID:        *msgID,
				ChatID:    *msgChatID,
				SenderID:  senderID,
				Content:   content,
				Type:      helpers.DerefString(messageType),
				CreatedAt: *createdAt,
			}
		}

		chats = append(chats, cp)
	}

	return chats, nil
}
