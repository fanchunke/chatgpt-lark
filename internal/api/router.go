package api

import (
	"net/http"

	"github.com/fanchunke/xgpt3"

	"github.com/fanchunke/chatgpt-lark/internal/middleware"

	config "github.com/fanchunke/chatgpt-lark/conf"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	sdkginext "github.com/larksuite/oapi-sdk-gin"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
)

type router struct {
	*gin.Engine
	cfg         *config.Config
	xgpt3Client *xgpt3.Client
	larkClient  *lark.Client
}

func NewRouter(cfg *config.Config, xgpt3Client *xgpt3.Client, larkClient *lark.Client) (http.Handler, error) {
	gin.SetMode(gin.ReleaseMode)
	e := gin.Default()
	pprof.Register(e, "debug/pprof")

	r := &router{Engine: e, cfg: cfg, xgpt3Client: xgpt3Client, larkClient: larkClient}
	r.Use(middleware.Logger())
	r.Use(middleware.URLHandler("url"))
	r.Use(middleware.MethodHandler("method"))
	r.Use(middleware.RequestIDHandler("requestId", "X-Request-Id"))
	r.Use(middleware.AccessHandler())
	r.GET("/healthz", r.Healthz)

	callback := NewCallbackHandler(cfg, r.xgpt3Client, r.larkClient)
	handler := dispatcher.NewEventDispatcher(r.cfg.Lark.VerificationToken, r.cfg.Lark.EventEncryptKey).OnP2MessageReceiveV1(callback.OnP2MessageReceiveV1)
	r.POST("/lark/receive", sdkginext.NewEventHandlerFunc(handler))
	return r, nil
}
