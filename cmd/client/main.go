package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	config "github.com/shellus/frp-daemon/pkg/client"
	"github.com/shellus/frp-daemon/pkg/frp"
	"github.com/shellus/frp-daemon/pkg/installer"
	"github.com/shellus/frp-daemon/pkg/types"
)

var configPath = "./client.yaml"
var instancesPath = "./instances.yaml"
var binDir = "./bin"
var instancesDir = "./instances"

var cfg *config.ClientConfig
var instancesFile *config.InstancesFile
var runner *frp.Runner

func StartFrpInstance(instance types.InstanceConfigLocal) (err error) {
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
	return
}

func StopFrpInstance(name string) (err error) {
	if err := runner.StopInstance(name); err != nil {
		log.Printf("停止实例 %s 失败: %v", name, err)
	}
	return
}

func main() {
	// 加载配置并上线MQTT
	var err error
	cfg, err = config.LoadClientConfig(configPath)
	if err != nil {
		log.Fatalf("加载客户端配置失败: %v", err)
	}
	client, err := config.NewClient(cfg.Client, cfg.Mqtt, Handler{})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	client.Start()

	// 加载实例配置
	instancesFile, err = config.LoadInstancesFile(instancesPath)
	if err != nil {
		log.Fatalf("加载实例配置失败: %v", err)
	}

	// 创建FRP运行器
	runner = frp.NewRunner()

	// 启动所有实例
	for _, localInstanceConfig := range instancesFile.Instances {
		if err := StartFrpInstance(localInstanceConfig); err != nil {
			log.Fatalf("启动实例 %s 失败: %v", localInstanceConfig.Name, err)
		}
		log.Printf("启动实例 %s 成功，进程ID: %d", localInstanceConfig.Name, runner.GetInstancePid(localInstanceConfig.Name))
	}

	log.Println("FRP守护进程启动...")

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	log.Println("收到关闭信号，开始优雅关闭...")

	// 优雅关闭所有实例
	if err := runner.Close(); err != nil {
		log.Printf("关闭FRP实例时发生错误: %v", err)
	}

	log.Println("FRP守护进程已关闭")
}
