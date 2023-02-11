package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/fanchunke1991/chatgpt-lark/ent/chatent"
	"github.com/fanchunke1991/chatgpt-lark/ent/chatent/message"
)

type MessageService struct {
	client *chatent.Client
}

func NewMessageService(client *chatent.Client) *MessageService {
	return &MessageService{
		client: client,
	}
}

func (s *MessageService) ListLatestMessagesWithSpouse(ctx context.Context, userId string, sessionId int, limit int) ([]*chatent.Message, error) {
	msgs, err := s.client.Message.
		Query().
		Where(message.SessionIDEQ(sessionId), message.FromUserIDEQ(userId), message.HasSpouse()).
		Order(chatent.Desc(message.FieldCreatedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query message failed: %w", err)
	}

	spouseMsgs, err := s.client.Message.
		Query().
		Where(message.SessionIDEQ(sessionId), message.ToUserIDEQ(userId), message.HasSpouse()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query spouse message failed: %w", err)
	}
	spouseMsgMap := make(map[int]*chatent.Message, 0)
	for _, m := range spouseMsgs {
		spouseMsgMap[m.SpouseID] = m
	}

	result := make([]*chatent.Message, 0)
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].CreatedAt.Sub(msgs[j].CreatedAt) < 0
	})
	for _, m := range msgs {
		spouse, ok := spouseMsgMap[m.ID]
		if ok {
			result = append(result, m, spouse)
		}
	}
	return result, nil
}

func (s *MessageService) CreateMessage(ctx context.Context, session *chatent.Session, fromUserId, toUserId, content string) (*chatent.Message, error) {
	r, err := s.client.Message.
		Create().
		SetSession(session).
		SetFromUserID(fromUserId).
		SetToUserID(toUserId).
		SetContent(content).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("Create Message failed: %w", err)
	}
	return r, nil
}

func (s *MessageService) CreateSpouseMessage(ctx context.Context, session *chatent.Session, fromUserId, toUserId, content string, spouse *chatent.Message) (*chatent.Message, error) {
	r, err := s.client.Message.
		Create().
		SetSession(session).
		SetFromUserID(fromUserId).
		SetToUserID(toUserId).
		SetContent(content).
		SetSpouse(spouse).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("Create Spouse Message failed: %w", err)
	}
	return r, nil
}
