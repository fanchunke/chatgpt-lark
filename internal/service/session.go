package service

import (
	"context"
	"fmt"

	"github.com/fanchunke/chatgpt-lark/ent/chatent"
	"github.com/fanchunke/chatgpt-lark/ent/chatent/session"
)

type SessionService struct {
	client *chatent.Client
}

func NewSessionService(client *chatent.Client) *SessionService {
	return &SessionService{
		client: client,
	}
}

func (s *SessionService) GetLatestActiveSession(ctx context.Context, userId string) (*chatent.Session, error) {
	result, err := s.client.Session.
		Query().
		Where(session.UserIDEQ(userId), session.StatusEQ(true)).
		Order(chatent.Desc(session.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetLatestActiveSession failed: %w", err)
	}
	return result, nil
}

func (s *SessionService) CreateSession(ctx context.Context, userId string) (*chatent.Session, error) {
	result, err := s.client.Session.
		Create().
		SetUserID(userId).
		SetStatus(true).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("Create Session failed: %w", err)
	}
	return result, nil
}
