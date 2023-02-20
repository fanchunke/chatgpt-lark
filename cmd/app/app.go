package main

import (
	"flag"

	config "github.com/fanchunke/chatgpt-lark/conf"
	"github.com/fanchunke/chatgpt-lark/internal/app"
	"github.com/fanchunke/chatgpt-lark/pkg/logger"
	"github.com/rs/zerolog/log"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	conf := flag.String("conf", "conf/online.conf", "配置文件")
	initEnt := flag.Bool("init-ent", false, "是否初始化数据库")
	flag.Parse()
	cfg, err := config.New(*conf)
	if err != nil {
		log.Fatal().Err(err).Msg("Config Load Failed")
	}
	logger.Configure(logger.Config{
		ConsoleLoggingEnabled: true,
		FileLoggingEnabled:    true,
		Level:                 cfg.Logger.Level,
		Filename:              cfg.Logger.Filename,
	})

	// 数据库迁移
	if *initEnt {
		if err := app.Migrate(cfg); err != nil {
			log.Fatal().Err(err).Msg("failed creating schema resources")
		}
		return
	}

	app.Run(cfg)
}
