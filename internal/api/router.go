package api

import (
	"net/http"

	"github.com/fanchunke/chatgpt-lark/ent/chatent"

	"github.com/fanchunke/chatgpt-lark/internal/middleware"

	config "github.com/fanchunke/chatgpt-lark/conf"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	sdkginext "github.com/larksuite/oapi-sdk-gin"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	gogpt "github.com/sashabaranov/go-gpt3"
)

type router struct {
	*gin.Engine
	cfg           *config.Config
	gptClient     *gogpt.Client
	larkClient    *lark.Client
	chatentClient *chatent.Client
}

func NewRouter(cfg *config.Config, gptClient *gogpt.Client, larkClient *lark.Client, chatentClient *chatent.Client) (http.Handler, error) {
	gin.SetMode(gin.ReleaseMode)
	e := gin.Default()
	pprof.Register(e, "debug/pprof")

	r := &router{Engine: e, cfg: cfg, gptClient: gptClient, larkClient: larkClient, chatentClient: chatentClient}
	r.Use(middleware.Logger())
	r.Use(middleware.URLHandler("url"))
	r.Use(middleware.MethodHandler("method"))
	r.Use(middleware.RequestIDHandler("requestId", "X-Request-Id"))
	r.Use(middleware.AccessHandler())
	r.GET("/healthz", r.Healthz)

	callback := NewCallbackHandler(r.gptClient, r.larkClient, chatentClient)
	handler := dispatcher.NewEventDispatcher(r.cfg.Lark.VerificationToken, r.cfg.Lark.EventEncryptKey).OnP2MessageReceiveV1(callback.OnP2MessageReceiveV1)
	r.POST("/lark/receive", sdkginext.NewEventHandlerFunc(handler))
	return r, nil
}
