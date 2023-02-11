package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fanchunke1991/chatgpt-lark/ent/chatent"
	"github.com/fanchunke1991/chatgpt-lark/internal/service"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/rs/zerolog/log"
	gogpt "github.com/sashabaranov/go-gpt3"
)

const (
	maxToken     = 4000
	sessionTurns = 10
)

type callbackHandler struct {
	gptClient      *gogpt.Client
	larkClient     *lark.Client
	messageService *service.MessageService
	sessionService *service.SessionService
}

func NewCallbackHandler(gptClient *gogpt.Client, larkClient *lark.Client, chatentClient *chatent.Client) *callbackHandler {
	return &callbackHandler{
		gptClient:      gptClient,
		larkClient:     larkClient,
		messageService: service.NewMessageService(chatentClient),
		sessionService: service.NewSessionService(chatentClient),
	}
}

// OnP2MessageReceiveV1: 机器人接收到用户发送的消息后触发此事件。
func (h *callbackHandler) OnP2MessageReceiveV1(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	log.Debug().Msgf("收到飞书消息: %+v", larkcore.Prettify(event))
	content, err := h.convertMessage(ctx, event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Convert lark msg error: %v", err)
		return err
	}

	appId := event.EventV2Base.Header.AppID
	openId := *event.Event.Sender.SenderId.OpenId

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error().Msgf("recovery from: %v", err)
			}
		}()

		if err := h.getGPTResponse(ctx, appId, openId, content); err != nil {
			log.Error().Err(err).Msgf("Get GPT Response error: %v", err)
		}
	}()

	return nil
}

func (h *callbackHandler) OnP2MessageReadV1(ctx context.Context, event *larkim.P2MessageReadV1) error {
	fmt.Println(larkcore.Prettify(event))
	fmt.Println(event.RequestId())
	return nil
}

func (h *callbackHandler) OnP2UserCreatedV3(ctx context.Context, event *larkcontact.P2UserCreatedV3) error {
	fmt.Println(larkcore.Prettify(event))
	fmt.Println(event.RequestId())
	return nil
}

func (h *callbackHandler) convertMessage(ctx context.Context, event *larkim.P2MessageReceiveV1) (string, error) {
	content, err := h.unmarshalLarkMessageContent(*event.Event.Message.Content)
	if err != nil {
		return "", fmt.Errorf("unmarshalLarkMessageContent failed: %w", err)
	}

	switch *event.Event.Message.MessageType {
	case "text":
		if text, ok := content["text"]; ok {
			return text.(string), nil
		}
	}
	return "", fmt.Errorf("UnSupported MsgType: %v", *event.Event.Message.MessageType)
}

func (h *callbackHandler) unmarshalLarkMessageContent(content string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *callbackHandler) getGPTResponse(ctx context.Context, appId, openId, content string) error {
	session, newQuery := h.buildSessionQuery(ctx, openId, sessionTurns, content)

	var err error
	// 如果没有 session，创建一个 session
	if session == nil {
		session, err = h.sessionService.CreateSession(ctx, openId)
		if err != nil {
			return err
		}
	}
	// 保存用户消息
	msg, err := h.messageService.CreateMessage(ctx, session, openId, appId, content)
	if err != nil {
		return err
	}

	if len(newQuery) > maxToken {
		newQuery = string([]rune(newQuery)[len(newQuery)-maxToken:])
	}
	log.Info().Msgf("GPT Raw Prompt: %s, New Prompt: %s", content, newQuery)

	// 获取 GPT 回复
	req := gogpt.CompletionRequest{
		Model:           gogpt.GPT3TextDavinci003,
		MaxTokens:       1500,
		Prompt:          newQuery,
		TopP:            1,
		Temperature:     0.9,
		PresencePenalty: 0.6,
		User:            openId,
	}
	resp, err := h.gptClient.CreateCompletion(ctx, req)
	if err != nil {
		return fmt.Errorf("CreateCompletion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("Empty GPT Choices")
	}

	reply := strings.TrimSpace(resp.Choices[0].Text)

	// 保存 GPT 回复
	_, err = h.messageService.CreateSpouseMessage(ctx, session, appId, openId, reply, msg)
	if err != nil {
		return err
	}

	sendContent, _ := json.Marshal(map[string]string{
		"text": reply,
	})
	log.Info().Msgf("Start Send GPT Response: %s", string(sendContent))

	_, err = h.larkClient.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeOpenId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			ReceiveId(openId).
			Content(string(sendContent)).
			Build()).
		Build())

	if err != nil {
		return fmt.Errorf("Send Lark Message failed: %w", err)
	}

	return nil
}

func (h *callbackHandler) buildSessionQuery(ctx context.Context, userId string, limit int, query string) (*chatent.Session, string) {
	session, err := h.sessionService.GetLatestActiveSession(ctx, userId)
	if err != nil {
		log.Warn().Err(err).Msgf("GetLatestActiveSession failed")
		return nil, query
	}
	msgs, err := h.messageService.ListLatestMessagesWithSpouse(ctx, userId, session.ID, limit)
	if err != nil {
		log.Error().Err(err).Msgf("GetLatestActiveSession failed")
		return session, query
	}

	result := ""
	for _, m := range msgs {
		if m.FromUserID == userId {
			result += fmt.Sprintf("Q: %s\n", m.Content)
		} else {
			result += fmt.Sprintf("A: %s\n", m.Content)
		}
	}
	result += fmt.Sprintf("Q: %s\nA: ", query)
	return session, result
}
