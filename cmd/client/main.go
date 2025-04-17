package main

import (
	"log"
	"time"

	config "github.com/shellus/frp-daemon/pkg/client"
	"github.com/shellus/frp-daemon/pkg/frp"
	"github.com/shellus/frp-daemon/pkg/installer"
)

var configPath = "./client.yaml"
var instancesPath = "./instances.yaml"
var binDir = "./bin"

func main() {
	// 加载配置并上线MQTT
	// cfg, err := config.LoadClientConfig(configPath)
	// if err != nil {
	// 	log.Fatalf("加载客户端配置失败: %v", err)
	// }

	// 加载实例配置
	instances, err := config.LoadInstancesFile(instancesPath)
	if err != nil {
		log.Fatalf("加载实例配置失败: %v", err)
	}

	// 创建FRP运行器
	runner := frp.NewRunner()

	// 启动所有实例
	for _, instance := range instances.Instances {
		var frpPath string
		frpPath, err = installer.IsFRPInstalled(binDir, instance.Version)
		if err != nil {
			log.Printf("FRP版本 %s 未安装，开始安装", instance.Version)
			frpPath, err = installer.EnsureFRPInstalled(binDir, instance.Version)
			if err != nil {
				log.Fatalf("安装FRP版本 %s 失败: %v", instance.Version, err)
			}
			log.Printf("FRP版本 %s 安装成功", instance.Version)
		}

		if err := runner.StartInstance(instance.Name, instance.Version, frpPath, instance.ConfigPath); err != nil {
			log.Printf("启动实例 %s 失败: %v", instance.Name, err)
		}
		log.Printf("启动实例 %s 成功", instance.Name)
	}

	log.Println("FRP守护进程启动...")

	// TODO runner应该Ctrl+c优雅退出，关闭runner所有子进程
	for {
		time.Sleep(10 * time.Second)
	}
}
