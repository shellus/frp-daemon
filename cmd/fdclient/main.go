package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	config "github.com/shellus/frp-daemon/pkg/fdclient"
	"github.com/shellus/frp-daemon/pkg/frp"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.DateTime,
		FormatTimestamp: func(i interface{}) string {
			return time.Now().Format(time.DateTime)
		},
	}).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	// 定义目录
	var baseDir string
	if os.Getenv("FRP_DAEMON_BASE_DIR") != "" {
		baseDir = os.Getenv("FRP_DAEMON_BASE_DIR")
	} else {
		var homeDir string
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			logger.Warn().Msgf("获取用户家目录失败，使用/root作为默认目录，error=%v", err)
			homeDir = "/root"
		}
		baseDir = filepath.Join(homeDir, ".frp-daemon")
	}

	var configFilePath = filepath.Join(baseDir, "client.yaml")
	var frpBinDir = filepath.Join(baseDir, "frpc")
	var frpcConfigDir = filepath.Join(baseDir, "config")

	// 加载配置并上线MQTT
	cfg, err := config.LoadClientConfig(configFilePath)
	if err != nil {
		logger.Fatal().Msgf("加载客户端配置失败，error=%v", err)
	}

	// 创建FRP运行器
	runner := frp.NewRunner(logger)

	// 创建客户端
	client, err := config.NewClient(cfg, runner, frpBinDir, frpcConfigDir, logger)
	if err != nil {
		logger.Fatal().Msgf("创建客户端失败，error=%v", err)
	}
	err = client.Start()
	if err != nil {
		logger.Fatal().Msgf("启动客户端失败，error=%v", err)
	}

	logger.Info().Msgf("客户端启动成功，%s[%s]，frp实例数=%d", cfg.ClientConfig.Client.Name, cfg.ClientConfig.Client.ClientId, len(cfg.ClientConfig.Instances))

	// 启动状态报告定时器
	statusTicker := time.NewTicker(time.Minute)
	go func() {
		// 立即上报一次状态
		if err := client.ReportStatus(); err != nil {
			logger.Error().Msgf("首次上报状态失败，error=%v", err)
		}

		// 然后开始定时上报
		for range statusTicker.C {
			if err := client.ReportStatus(); err != nil {
				logger.Error().Msgf("上报状态失败，error=%v", err)
			}
		}
	}()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	logger.Info().Msg("收到关闭信号，开始优雅关闭...")

	// 关闭状态报告定时器
	statusTicker.Stop()

	// 优雅关闭所有实例
	if err := client.Stop(); err != nil {
		logger.Error().Msgf("关闭FRP实例时发生错误，error=%v", err)
	}

	logger.Info().Msg("FRP守护进程已关闭")
}
