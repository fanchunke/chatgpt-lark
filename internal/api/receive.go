package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	config "github.com/fanchunke/chatgpt-lark/conf"
	"github.com/fanchunke/xgpt3"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"
)

type versionType string

const (
	callbackVersionV1 versionType = "v1"
	callbackVersionV2 versionType = "v2"
)

type callbackHandler struct {
	cfg         *config.Config
	xgpt3Client *xgpt3.Client
	larkClient  *lark.Client
	version     versionType
}

func NewCallbackHandler(cfg *config.Config, xgpt3Client *xgpt3.Client, larkClient *lark.Client, version versionType) *callbackHandler {
	return &callbackHandler{
		cfg:         cfg,
		larkClient:  larkClient,
		xgpt3Client: xgpt3Client,
		version:     version,
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

		var reply string

		// 判断是否需要重启会话
		closeSession := h.cfg.Conversation.CloseSessionFlag == content

		// 获取回复
		if !closeSession {
			var handler func(ctx context.Context, appId, userId string, content string) (string, error)
			if h.version == callbackVersionV1 {
				handler = h.getOpenAICompletion
			} else {
				handler = h.getOpenAIChatCompletion
			}

			reply, err = handler(context.Background(), appId, openId, content)
			if err != nil {
				log.Error().Err(err).Msgf("Get GPT Response error: %v", err)
				return
			}
		} else {
			if err := h.xgpt3Client.CloseConversation(context.Background(), openId); err != nil {
				log.Error().Err(err).Msgf("Close Conversation error: %v", err)
				return
			}
			reply = h.cfg.Conversation.CloseSessionReply
		}

		// 发送回复
		if reply == "" {
			log.Debug().Msg("Reply is empty")
			return
		}
		if err := h.sendTextMessage(context.Background(), appId, openId, reply); err != nil {
			log.Error().Err(err).Msgf("Send Lark Response error: %v", err)
			return
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

func (h *callbackHandler) sendTextMessage(ctx context.Context, appId, userId, content string) error {
	sendContent, _ := json.Marshal(map[string]string{
		"text": content,
	})
	log.Info().Msgf("[AppId: %d] [UserId: %s] Start Send Lark Response: %s", appId, userId, string(sendContent))
	_, err := h.larkClient.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeOpenId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			ReceiveId(userId).
			Content(string(sendContent)).
			Build()).
		Build())

	if err != nil {
		return fmt.Errorf("Send Lark Message failed: %w", err)
	}

	return nil
}

func (h *callbackHandler) getOpenAICompletion(ctx context.Context, appId, userId, content string) (string, error) {
	// 获取 GPT 回复
	req := openai.CompletionRequest{
		Model:           openai.GPT3TextDavinci003,
		MaxTokens:       1500,
		Prompt:          content,
		TopP:            1,
		Temperature:     0.9,
		PresencePenalty: 0.6,
		User:            userId,
	}

	var resp openai.CompletionResponse
	var err error
	if h.cfg.Conversation.EnableConversation {
		resp, err = h.xgpt3Client.CreateConversationCompletionWithChannel(ctx, req, appId)
	} else {
		resp, err = h.xgpt3Client.Client.CreateCompletion(ctx, req)
	}

	if err != nil {
		return "", fmt.Errorf("CreateCompletion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("Empty GPT Choices")
	}

	// 发送回复给用户
	reply := strings.TrimSpace(resp.Choices[0].Text)
	return reply, nil
}

func (h *callbackHandler) getOpenAIChatCompletion(ctx context.Context, appId, userId, content string) (string, error) {
	// 获取 GPT 回复
	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: 1500,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: content,
			},
		},
		TopP:            1,
		Temperature:     0.9,
		PresencePenalty: 0.6,
		User:            userId,
	}
	var resp openai.ChatCompletionResponse
	var err error
	if h.cfg.Conversation.EnableConversation {
		resp, err = h.xgpt3Client.CreateChatCompletionWithChannel(ctx, req, appId)
	} else {
		resp, err = h.xgpt3Client.Client.CreateChatCompletion(ctx, req)
	}

	if err != nil {
		return "", fmt.Errorf("CreateCompletion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("Empty GPT Choices")
	}

	// 发送回复给用户
	reply := strings.TrimSpace(resp.Choices[0].Message.Content)
	return reply, nil
}
