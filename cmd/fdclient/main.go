package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	config "github.com/shellus/frp-daemon/pkg/fdclient"
	"github.com/shellus/frp-daemon/pkg/frp"
)

func main() {

	// 定义目录
	var baseDir string
	if os.Getenv("FRP_DAEMON_BASE_DIR") != "" {
		baseDir = os.Getenv("FRP_DAEMON_BASE_DIR")
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("获取用户家目录失败: %v", err)
		}
		baseDir = filepath.Join(homeDir, ".frp-daemon")
	}

	var configFilePath = filepath.Join(baseDir, "client.yaml")
	var frpBinDir = filepath.Join(baseDir, "frpc")
	var frpcConfigDir = filepath.Join(baseDir, "config")

	// 加载配置并上线MQTT
	cfg, err := config.LoadClientConfig(configFilePath)
	if err != nil {
		log.Fatalf("加载客户端配置失败: %v", err)
	}

	// 创建FRP运行器
	runner := frp.NewRunner()

	// 创建客户端
	client, err := config.NewClient(cfg, runner, frpBinDir, frpcConfigDir)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	client.Start()

	log.Printf("客户端 %s[%s] 启动成功, frp实例数：%d", cfg.ClientConfig.Client.Name, cfg.ClientConfig.Client.ClientId, len(cfg.ClientConfig.Instances))

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	log.Println("收到关闭信号，开始优雅关闭...")

	// 优雅关闭所有实例
	if err := client.Stop(); err != nil {
		log.Printf("关闭FRP实例时发生错误: %v", err)
	}

	log.Println("FRP守护进程已关闭")
}
