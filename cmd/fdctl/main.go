package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/shellus/frp-daemon/pkg/emqx"
	cl "github.com/shellus/frp-daemon/pkg/fdclient"
	"github.com/shellus/frp-daemon/pkg/fdctl"
	"github.com/shellus/frp-daemon/pkg/types"
	"gopkg.in/yaml.v3"
)

// 全局配置路径
var configFilePath string

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

	configFilePath = filepath.Join(baseDir, "controller.yaml")

	// 加载控制端配置
	cfg, err := fdctl.LoadControllerConfig(configFilePath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 检查子命令
	if len(os.Args) < 2 {
		runController()
		return
	}

	// 处理子命令
	switch os.Args[1] {
	case "new":
		handleNewCmd(cfg)
	case "delete":
		handleDeleteCmd(cfg)
	case "update":
		handleUpdateCmd(cfg)
	case "ping":
		handlePingCmd(cfg)
	default:
		log.Fatalf("未知命令: %s", os.Args[1])
	}
}

// 处理new子命令
func handleNewCmd(cfg *fdctl.ControllerConfig) {
	// 创建new子命令
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	clientName := newCmd.String("name", "", "客户端名称")

	// 解析子命令参数
	if err := newCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *clientName == "" {
		log.Fatal("请使用 -name 参数指定客户端名称")
	}

	// 创建EMQX API客户端
	api := emqx.NewAPI(cfg.EMQXAPI)

	// 创建新客户端
	auth := &types.ClientAuth{
		Name:     *clientName,
		ClientId: types.GenerateRandomString(16),
		Password: types.GenerateRandomString(32),
	}

	mqttconect, err := api.CreateUser(auth)
	if err != nil {
		log.Fatalf("创建MQTT用户失败: %v", err)
	}

	// 将结构体转换为YAML格式
	yamlStr, err := yaml.Marshal(&cl.ClientConfig{
		Client: *auth,
		Mqtt:   *mqttconect,
	})
	if err != nil {
		log.Fatalf("转换客户端配置到YAML失败: %v", err)
	}

	log.Printf("生成客户端成功\n# 请将以下内容保存到client.yaml文件中，作为被控端的配置，本配置中的mqtt部分只会出现一次\n%s", string(yamlStr))

	cfg.Clients = append(cfg.Clients, *auth)
	if err := fdctl.WriteControllerConfig(*cfg, configFilePath); err != nil {
		log.Fatalf("写入控制器配置失败: %v", err)
	}
	log.Printf("被控端配置写入成功")
}

// 处理delete子命令
func handleDeleteCmd(cfg *fdctl.ControllerConfig) {
	// 创建delete子命令
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteClientName := deleteCmd.String("name", "", "要删除的客户端名称")

	// 解析delete子命令参数
	if err := deleteCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *deleteClientName == "" {
		log.Fatal("请使用 -name 参数指定要删除的客户端名称")
	}

	// 查找要删除的客户端
	var clientIndex = -1
	var clientToDelete *types.ClientAuth
	for i, client := range cfg.Clients {
		if client.Name == *deleteClientName {
			clientIndex = i
			clientToDelete = &client
			break
		}
	}

	if clientIndex == -1 {
		log.Fatalf("未找到名为 %s 的客户端", *deleteClientName)
	}

	// 创建EMQX API客户端
	api := emqx.NewAPI(cfg.EMQXAPI)

	// 删除MQTT用户
	if err := api.DeleteUser(clientToDelete); err != nil {
		log.Printf("删除MQTT用户失败: %v", err)
		return
	}

	// 从配置中删除客户端
	cfg.Clients = append(cfg.Clients[:clientIndex], cfg.Clients[clientIndex+1:]...)

	// 保存更新后的配置
	if err := fdctl.WriteControllerConfig(*cfg, configFilePath); err != nil {
		log.Fatalf("写入控制器配置失败: %v", err)
	}

	log.Printf("成功删除客户端: %s", *deleteClientName)
}

// createController 创建并连接控制器
func createController(cfg *fdctl.ControllerConfig) (*fdctl.Controller, error) {
	// 创建控制器实例
	ctrl, err := fdctl.NewController(cfg.Client, cfg.MQTT)
	if err != nil {
		return nil, fmt.Errorf("创建控制器失败: %v", err)
	}

	// 连接MQTT
	if err := ctrl.ConnectMQTT(); err != nil {
		return nil, fmt.Errorf("连接MQTT失败: %v", err)
	}

	return ctrl, nil
}

// 处理update子命令
func handleUpdateCmd(cfg *fdctl.ControllerConfig) {
	// 创建update子命令
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	updateClientName := updateCmd.String("name", "", "客户端名称")
	instanceName := updateCmd.String("instance", "", "实例名称")
	frpVersion := updateCmd.String("version", "", "frp版本")
	configFile := updateCmd.String("config", "", "配置文件路径")

	// 解析update子命令参数
	if err := updateCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *updateClientName == "" {
		log.Fatal("请使用 -name 参数指定客户端名称")
	}
	if *instanceName == "" {
		log.Fatal("请使用 -instance 参数指定实例名称")
	}
	if *frpVersion == "" {
		log.Fatal("请使用 -version 参数指定frp版本")
	}
	if *configFile == "" {
		log.Fatal("请使用 -config 参数指定配置文件路径")
	}

	// 创建控制器
	ctrl, err := createController(cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// 创建配置对象
	config := &types.InstanceConfigLocal{
		Name:       *instanceName,
		Version:    *frpVersion,
		ConfigPath: *configFile,
	}

	// 发送配置
	if err := ctrl.SendConfig(*updateClientName, config); err != nil {
		log.Fatalf("发送配置失败: %v", err)
	}

	log.Printf("配置已成功发送到客户端: %s", *updateClientName)
}

// 处理ping子命令
func handlePingCmd(cfg *fdctl.ControllerConfig) {
	// 创建ping子命令
	pingCmd := flag.NewFlagSet("ping", flag.ExitOnError)
	pingClientName := pingCmd.String("name", "", "客户端名称")

	// 解析ping子命令参数
	if err := pingCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *pingClientName == "" {
		log.Fatal("请使用 -name 参数指定客户端名称")
	}

	// 创建控制器
	ctrl, err := createController(cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// 发送ping消息
	if err := ctrl.SendPing(*pingClientName); err != nil {
		log.Fatalf("发送ping消息失败: %v", err)
	}

	log.Printf("已向客户端 %s 发送ping消息", *pingClientName)
}

func runController() {
	log.Println("FRP控制器启动...")
}
