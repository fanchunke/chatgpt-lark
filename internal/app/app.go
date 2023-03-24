package app

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/fanchunke/xgpt3"
	"github.com/fanchunke/xgpt3/conversation/ent"
	"github.com/fanchunke/xgpt3/conversation/ent/chatent"

	config "github.com/fanchunke/chatgpt-lark/conf"
	"github.com/fanchunke/chatgpt-lark/internal/api"
	"github.com/fanchunke/chatgpt-lark/pkg/httpserver"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"
)

func Run(cfg *config.Config) {
	log.Info().Msgf("Config: %v", cfg)

	// 初始化 gpt client
	gptClient := openai.NewClient(cfg.GPT.ApiKey)

	// 初始化 lark client
	larkClient := lark.NewClient(
		cfg.Lark.AppId,
		cfg.Lark.AppSecret,
		lark.WithOpenBaseUrl(cfg.Lark.BaseUrl),
		lark.WithLogLevel(larkcore.LogLevelDebug),
	)

	// 初始化数据库 client
	dbConf := cfg.Database
	chatentClient, err := chatent.Open(dbConf.Driver, dbConf.DataSource)
	if err != nil {
		log.Fatal().Err(err).Msg("ent - open database failed")
	}
	if err := Migrate(cfg); err != nil {
		log.Fatal().Err(err).Msg("ent - database migrate failed")
	}
	log.Info().Msg("数据库迁移成功")

	// 初始化 xgpt3 client
	xgpt3Client := xgpt3.NewClient(gptClient, ent.New(chatentClient))

	handler, err := api.NewRouter(cfg, xgpt3Client, larkClient)
	if err != nil {
		log.Fatal().Err(err).Msg("api - Router - api.Router failed")
	}
	httpServer := httpserver.New(handler, httpserver.Port(cfg.HTTP.Port))
	httpServer.Start()
	log.Info().Msg("Server Started")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		log.Info().Msgf("app - Run - signal: %s", s.String())
	case err := <-httpServer.Notify():
		log.Error().Err(err).Msg("app - Run - httpServer.Notify")
	}

	err = httpServer.Shutdown()
	if err != nil {
		log.Error().Err(err).Msg("app - Run - httpServer.Shutdown")
	}

}
