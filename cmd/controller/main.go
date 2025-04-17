package main

import (
	"flag"
	"log"

	"github.com/shellus/frp-daemon/pkg/controller"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "c", "config.yaml", "配置文件路径")
	flag.Parse()
}

func main() {
	// 加载配置
	_, err := controller.LoadControllerConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Println("FRP控制器启动...")
	// TODO: 初始化MQTT客户端
	// TODO: 实现命令处理: update <client> <instance> <config>, status <client> <instance>
	// TODO: 实现状态监控: tail <client> <instance>
	// TODO: 实现新建配置: new ？？？
}
