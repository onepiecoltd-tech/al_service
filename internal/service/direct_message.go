package service

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

const directMessageHistoryLimit = 200

type DirectMessageService interface {
	// History returns the conversation between the caller and otherID — only
	// allowed between accepted friends.
	History(ctx context.Context, userID, otherID uuid.UUID) ([]model.DirectMessage, error)
	Send(ctx context.Context, senderID, receiverID uuid.UUID, body string) (*model.DirectMessage, error)
	// Subscribe registers a channel that receives every message addressed to
	// userID as it's sent, for the WebSocket stream endpoint. unsubscribe
	// must be called (e.g. via defer) when the connection closes.
	Subscribe(userID uuid.UUID) (ch <-chan model.DirectMessage, unsubscribe func())
}

type directMessageService struct {
	messages repository.DirectMessageRepository
	users    repository.UserRepository

	mu   sync.Mutex
	subs map[uuid.UUID][]chan model.DirectMessage
}

func NewDirectMessageService(messages repository.DirectMessageRepository, users repository.UserRepository) DirectMessageService {
	return &directMessageService{
		messages: messages,
		users:    users,
		subs:     make(map[uuid.UUID][]chan model.DirectMessage),
	}
}

func (s *directMessageService) History(ctx context.Context, userID, otherID uuid.UUID) ([]model.DirectMessage, error) {
	ok, err := s.users.AreFriends(ctx, userID, otherID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("chỉ có thể xem hội thoại với bạn bè")
	}
	return s.messages.ListConversation(ctx, userID, otherID, directMessageHistoryLimit)
}

func (s *directMessageService) Send(ctx context.Context, senderID, receiverID uuid.UUID, body string) (*model.DirectMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, apperror.BadRequest("tin nhắn không được để trống")
	}
	ok, err := s.users.AreFriends(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("chỉ có thể nhắn tin với bạn bè")
	}
	msg, err := s.messages.Insert(ctx, senderID, receiverID, body)
	if err != nil {
		return nil, err
	}
	s.publish(receiverID, *msg)
	return msg, nil
}

func (s *directMessageService) Subscribe(userID uuid.UUID) (<-chan model.DirectMessage, func()) {
	ch := make(chan model.DirectMessage, 8)
	s.mu.Lock()
	s.subs[userID] = append(s.subs[userID], ch)
	s.mu.Unlock()

	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		chs := s.subs[userID]
		for i, c := range chs {
			if c == ch {
				s.subs[userID] = append(chs[:i], chs[i+1:]...)
				break
			}
		}
		close(ch)
	}
	return ch, unsubscribe
}

// publish is best-effort: if the receiver isn't currently subscribed (not on
// the page), the message just waits in the DB for their next history fetch.
func (s *directMessageService) publish(receiverID uuid.UUID, msg model.DirectMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subs[receiverID] {
		select {
		case ch <- msg:
		default: // subscriber's buffer is full — drop rather than block Send
		}
	}
}
