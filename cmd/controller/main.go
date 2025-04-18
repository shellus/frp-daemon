package main

import (
	"flag"
	"log"
	"os"

	cl "github.com/shellus/frp-daemon/pkg/client"
	"github.com/shellus/frp-daemon/pkg/controller"
	"github.com/shellus/frp-daemon/pkg/emqx"
	"github.com/shellus/frp-daemon/pkg/types"
	"gopkg.in/yaml.v3"
)

const configPath = "./controller.yaml"

func main() {
	// TODO: 初始化MQTT客户端
	// TODO: 实现命令处理: update <client> <instance> <config>, status <client> <instance>
	// TODO: 实现状态监控: tail <client> <instance>
	// TODO: 实现新建配置: new ？？？

	// 加载控制端配置
	cfg, err := controller.LoadControllerConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建new子命令
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	clientName := newCmd.String("name", "", "客户端名称")

	// 解析主命令参数
	flag.Parse()

	// 检查子命令
	if len(os.Args) < 2 {
		runController()
		return
	}

	// 处理子命令
	switch os.Args[1] {
	case "new":
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
		if err := controller.WriteControllerConfig(cfg, configPath); err != nil {
			log.Fatalf("写入控制器配置失败: %v", err)
		}
		log.Printf("被控端配置写入成功")

	default:
		log.Fatalf("未知命令: %s", os.Args[1])
	}
}

func runController() {
	log.Println("FRP控制器启动...")
}
