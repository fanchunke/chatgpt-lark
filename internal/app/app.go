package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fanchunke1991/chatgpt-lark/ent/chatent"

	config "github.com/fanchunke1991/chatgpt-lark/conf"
	"github.com/fanchunke1991/chatgpt-lark/internal/api"
	"github.com/fanchunke1991/chatgpt-lark/pkg/httpserver"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/rs/zerolog/log"
	gogpt "github.com/sashabaranov/go-gpt3"
)

func Run(cfg *config.Config) {
	log.Info().Msgf("Config: %v", cfg)

	// 初始化 gpt client
	gptClient := gogpt.NewClient(cfg.GPT.ApiKey)

	// 初始化 lark client
	larkClient := lark.NewClient(
		cfg.Lark.AppId,
		cfg.Lark.AppSecret,
		lark.WithOpenBaseUrl(cfg.Lark.BaseUrl),
		lark.WithLogLevel(larkcore.LogLevelDebug),
	)

	// 初始化数据库 client
	dbConf := cfg.Database
	s := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=True", dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.DBName)
	chatentClient, err := chatent.Open(dbConf.Dialect, s)
	if err != nil {
		log.Fatal().Err(err).Msg("ent - open database failed")
	}

	handler, err := api.NewRouter(cfg, gptClient, larkClient, chatentClient)
	if err != nil {
		log.Fatal().Err(err).Msg("api - Router - api.Router failed")
	}
	httpServer := httpserver.New(handler, httpserver.Port(cfg.HTTP.Port))
	httpServer.Start()
	log.Info().Msg("Server Started")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

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
