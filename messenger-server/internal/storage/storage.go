package storage

import "errors"

var (
	ErrUserExists = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
	ErrChatExists = errors.New("chat already exists")
	ErrChatNotFound = errors.New("chat not found")
	ErrChatOrUserNotFound = errors.New(ErrUserNotFound.Error() + ", " + ErrChatNotFound.Error())
	ErrUserIsAlreadyInChat = errors.New("user is already in chat")
	ErrNoMembersOfTheChat = errors.New("no members of the chat")
	ErrEmptyMemberInput = errors.New("empty member input")
	ErrInvalidRole = errors.New("invalid role")
	ErrDuplicateMember = errors.New("duplicate member")
	ErrEmptyUserIDs = errors.New("empty user IDs")
	ErrNoMembersRemoved = errors.New("no members removed")
	ErrUserIsNotMember = errors.New("user is not a member")
	ErrMessageNotFound = errors.New("message not found")
)