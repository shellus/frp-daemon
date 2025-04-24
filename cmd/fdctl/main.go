package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/shellus/frp-daemon/pkg/emqx"
	cl "github.com/shellus/frp-daemon/pkg/fdclient"
	"github.com/shellus/frp-daemon/pkg/fdctl"
	"github.com/shellus/frp-daemon/pkg/types"
	"gopkg.in/yaml.v3"
)

var logger zerolog.Logger

// 全局配置路径
var configFilePath string

func main() {
	logger = zerolog.New(zerolog.ConsoleWriter{
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
		homeDir, err := os.UserHomeDir()
		if err != nil {
			logger.Fatal().Msgf("获取用户家目录失败: %v", err)
		}
		baseDir = filepath.Join(homeDir, ".frp-daemon")
	}

	configFilePath = filepath.Join(baseDir, "controller.yaml")

	// 加载控制端配置
	cfg, err := fdctl.LoadControllerConfig(configFilePath)
	if err != nil {
		logger.Fatal().Msgf("加载配置失败: %v", err)
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
	case "status":
		handleStatusCmd(cfg)
	case "wol":
		handleWOLCmd(cfg)
	default:
		logger.Fatal().Msgf("未知命令: %s", os.Args[1])
	}
}

// 处理new子命令
func handleNewCmd(cfg *fdctl.ControllerConfig) {
	// 创建new子命令
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	clientName := newCmd.String("name", "", "客户端名称")

	// 解析子命令参数
	if err := newCmd.Parse(os.Args[2:]); err != nil {
		logger.Fatal().Msgf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *clientName == "" {
		logger.Fatal().Msg("请使用 -name 参数指定客户端名称")
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
		logger.Fatal().Msgf("创建MQTT用户失败: %v", err)
	}

	// 将结构体转换为YAML格式
	yamlStr, err := yaml.Marshal(&cl.ClientConfig{
		Client: *auth,
		Mqtt:   *mqttconect,
	})
	if err != nil {
		logger.Fatal().Msgf("转换客户端配置到YAML失败: %v", err)
	}

	logger.Info().Msgf("生成客户端成功\n# 请将以下内容保存到client.yaml文件中，作为被控端的配置，本配置中的mqtt部分只会出现一次\n%s", string(yamlStr))

	cfg.Clients = append(cfg.Clients, *auth)
	if err := fdctl.WriteControllerConfig(*cfg, configFilePath); err != nil {
		logger.Fatal().Msgf("写入控制器配置失败: %v", err)
	}
	logger.Info().Msg("被控端配置写入成功")
}

// 处理delete子命令
func handleDeleteCmd(cfg *fdctl.ControllerConfig) {
	// 创建delete子命令
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteClientName := deleteCmd.String("name", "", "客户端名称")
	deleteInstanceName := deleteCmd.String("instance", "", "要删除的实例名称")

	// 解析delete子命令参数
	if err := deleteCmd.Parse(os.Args[2:]); err != nil {
		logger.Fatal().Msgf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *deleteClientName == "" {
		logger.Fatal().Msg("请使用 -name 参数指定客户端名称")
	}
	if *deleteInstanceName == "" {
		logger.Fatal().Msg("请使用 -instance 参数指定要删除的实例名称")
	}

	// 查找要删除的客户端
	var clientToDelete *types.ClientAuth
	for _, client := range cfg.Clients {
		if client.Name == *deleteClientName {
			clientToDelete = &client
			break
		}
	}

	if clientToDelete == nil {
		logger.Fatal().Msgf("未找到名为 %s 的客户端", *deleteClientName)
	}

	// 创建控制器
	ctrl, err := createController(cfg)
	if err != nil {
		logger.Fatal().Msgf("创建控制器失败: %v", err)
	}
	defer ctrl.MqttClient.Disconnect()

	// 删除实例
	if err := ctrl.DeleteInstance(clientToDelete.ClientId, *deleteInstanceName); err != nil {
		logger.Fatal().Msgf("删除实例失败: %v", err)
	}

	logger.Info().Msgf("成功删除实例: %s", *deleteInstanceName)
}

// createController 创建并连接控制器
func createController(cfg *fdctl.ControllerConfig) (*fdctl.Controller, error) {
	// 创建控制器实例
	ctrl, err := fdctl.NewController(cfg.Client, cfg.MQTT, logger)
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
		logger.Fatal().Msgf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *updateClientName == "" {
		logger.Fatal().Msg("请使用 -name 参数指定客户端名称")
	}
	if *instanceName == "" {
		logger.Fatal().Msg("请使用 -instance 参数指定实例名称")
	}
	if *frpVersion == "" {
		logger.Fatal().Msg("请使用 -version 参数指定frp版本")
	}
	if *configFile == "" {
		logger.Fatal().Msg("请使用 -config 参数指定配置文件路径")
	}

	// 名称转为clientId
	var targetClient *types.ClientAuth
	for _, client := range cfg.Clients {
		if client.Name == *updateClientName {
			targetClient = &client
			break
		}
	}

	if targetClient == nil {
		logger.Fatal().Msgf("未找到名为 %s 的客户端", *updateClientName)
	}

	// 创建控制器
	ctrl, err := createController(cfg)
	if err != nil {
		logger.Fatal().Msgf("%v", err)
	}

	// 创建配置对象
	config := types.InstanceConfigLocal{
		Name:       *instanceName,
		Version:    *frpVersion,
		ConfigPath: *configFile,
	}

	// 发送配置
	if err := ctrl.SendConfig(targetClient.ClientId, targetClient.Password, config); err != nil {
		logger.Fatal().Msgf("发送配置失败: %v", err)
	}

	// 下发不代表客户端已经收到
	logger.Info().Msgf("配置已成功下发: %s[%s]", *updateClientName, targetClient.ClientId)
}

// 处理ping子命令
func handlePingCmd(cfg *fdctl.ControllerConfig) {
	// 创建ping子命令
	pingCmd := flag.NewFlagSet("ping", flag.ExitOnError)
	pingClientName := pingCmd.String("name", "", "客户端名称")

	// 解析ping子命令参数
	if err := pingCmd.Parse(os.Args[2:]); err != nil {
		logger.Fatal().Msgf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *pingClientName == "" {
		logger.Fatal().Msg("请使用 -name 参数指定客户端名称")
	}

	// 名称转为clientId
	var targetClient *types.ClientAuth
	for _, client := range cfg.Clients {
		if client.Name == *pingClientName {
			targetClient = &client
			break
		}
	}

	if targetClient == nil {
		logger.Fatal().Msgf("未找到名为 %s 的客户端", *pingClientName)
	}

	// 创建控制器
	ctrl, err := createController(cfg)
	if err != nil {
		logger.Fatal().Msgf("%v", err)
	}

	// 发送ping消息
	if err := ctrl.SendPing(targetClient.ClientId); err != nil {
		logger.Fatal().Msgf("发送ping消息失败: %v", err)
	}

	logger.Info().Msgf("已向客户端 %s[%s] 发送ping消息", *pingClientName, targetClient.ClientId)
}

// 处理status子命令
func handleStatusCmd(cfg *fdctl.ControllerConfig) {
	// 创建status子命令
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	statusClientName := statusCmd.String("name", "", "客户端名称")
	statusInstanceName := statusCmd.String("instance", "", "实例名称")

	// 解析status子命令参数
	if err := statusCmd.Parse(os.Args[2:]); err != nil {
		logger.Fatal().Msgf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *statusClientName == "" {
		logger.Fatal().Msg("请使用 -name 参数指定客户端名称")
	}
	if *statusInstanceName == "" {
		logger.Fatal().Msg("请使用 -instance 参数指定实例名称")
	}

	// 查找客户端
	var clientToQuery *types.ClientAuth
	for _, client := range cfg.Clients {
		if client.Name == *statusClientName {
			clientToQuery = &client
			break
		}
	}

	if clientToQuery == nil {
		logger.Fatal().Msgf("未找到名为 %s 的客户端", *statusClientName)
	}

	// 创建控制器
	ctrl, err := createController(cfg)
	if err != nil {
		logger.Fatal().Msgf("创建控制器失败: %v", err)
	}
	defer ctrl.MqttClient.Disconnect()

	// 获取状态
	status, err := ctrl.GetStatus(clientToQuery.ClientId, *statusInstanceName)
	if err != nil {
		logger.Fatal().Msgf("获取状态失败: %v", err)
	}

	// 打印状态
	logger.Info().Msgf("实例状态: %+v", status)
}

// 处理wol子命令
func handleWOLCmd(cfg *fdctl.ControllerConfig) {
	// 创建wol子命令
	wolCmd := flag.NewFlagSet("wol", flag.ExitOnError)
	wolClientName := wolCmd.String("name", "", "客户端名称")
	macAddress := wolCmd.String("mac", "", "目标设备的MAC地址")

	// 解析wol子命令参数
	if err := wolCmd.Parse(os.Args[2:]); err != nil {
		logger.Fatal().Msgf("解析参数失败: %v", err)
	}

	// 检查必需参数
	if *wolClientName == "" {
		logger.Fatal().Msg("请使用 -name 参数指定客户端名称")
	}
	if *macAddress == "" {
		logger.Fatal().Msg("请使用 -mac 参数指定目标设备的MAC地址")
	}

	// 查找客户端
	var clientToWake *types.ClientAuth
	for _, client := range cfg.Clients {
		if client.Name == *wolClientName {
			clientToWake = &client
			break
		}
	}

	if clientToWake == nil {
		logger.Fatal().Msgf("未找到名为 %s 的客户端", *wolClientName)
	}

	// 创建控制器
	ctrl, err := createController(cfg)
	if err != nil {
		logger.Fatal().Msgf("创建控制器失败: %v", err)
	}
	defer ctrl.MqttClient.Disconnect()

	// 发送WOL命令
	if err := ctrl.SendWOL(clientToWake.ClientId, *macAddress); err != nil {
		logger.Fatal().Msgf("发送WOL命令失败: %v", err)
	}

	logger.Info().Msgf("已向客户端 %s[%s] 发送WOL命令，目标MAC地址: %s", *wolClientName, clientToWake.ClientId, *macAddress)
}

func runController() {
	logger.Info().Msg("FRP控制器启动...")
}
