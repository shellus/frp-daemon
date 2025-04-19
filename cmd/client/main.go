package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	config "github.com/shellus/frp-daemon/pkg/client"
	"github.com/shellus/frp-daemon/pkg/frp"
)

var configPath = "./client.yaml"
var instancesPath = "./instances.yaml"
var binDir = "./bin"
var instancesDir = "./instances"

func main() {
	// 加载配置并上线MQTT
	var err error
	cfg, err := config.LoadClientConfig(configPath)
	if err != nil {
		log.Fatalf("加载客户端配置失败: %v", err)
	}

	// 加载实例配置
	instancesFile, err := config.LoadInstancesFile(instancesPath)
	if err != nil {
		log.Fatalf("加载实例配置失败: %v", err)
	}

	// 创建FRP运行器
	runner := frp.NewRunner()

	// 创建客户端
	client, err := config.NewClient(cfg.Client, cfg.Mqtt, instancesFile, runner, binDir, instancesDir)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	client.Start()

	log.Println("FRP守护进程启动...")

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
