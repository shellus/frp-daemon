package fdclient

import (
	"log"

	"github.com/shellus/frp-daemon/pkg/frp"
	"github.com/shellus/frp-daemon/pkg/installer"
	mqttC "github.com/shellus/frp-daemon/pkg/mqtt"
	"github.com/shellus/frp-daemon/pkg/types"
)

type Client struct {
	auth          types.ClientAuth
	mqttConfig    types.MQTTClientOpts
	mqtt          *mqttC.MQTT
	instancesFile *InstancesFile
	runner        *frp.Runner
	binDir        string
	instancesDir  string
}

func NewClient(auth types.ClientAuth, mqttConfig types.MQTTClientOpts, instancesFile *InstancesFile, runner *frp.Runner, binDir, instancesDir string) (*Client, error) {
	client := &Client{
		auth:          auth,
		mqttConfig:    mqttConfig,
		instancesFile: instancesFile,
		runner:        runner,
		binDir:        binDir,
		instancesDir:  instancesDir,
	}

	mqtt := mqttC.NewMQTT(mqttConfig)
	if err := mqtt.Connect(); err != nil {
		return nil, err
	}

	client.mqtt = mqtt
	client.mqtt.Subscribe(mqttC.MessageTopic(mqttConfig.TopicPrefix, auth.ClientId), byte(mqttConfig.QoS), func(message types.Message) {
		// 判断消息action
		switch message.Action {
		case types.MessageActionPing:
			client.HandlePing(message)
		case types.MessageActionUpdate:
			client.HandleUpdate(message)
		default:
			log.Printf("未处理的消息动作: action=%s, payload=%s", message.Action, message.Payload)
		}
	})

	return client, nil
}

func (c *Client) Start() {
	for _, localInstanceConfig := range c.instancesFile.Instances {
		if err := c.StartFrpInstance(localInstanceConfig); err != nil {
			log.Fatalf("启动实例 %s 失败: %v", localInstanceConfig.Name, err)
		}
		log.Printf("启动实例 %s 成功，进程ID: %d", localInstanceConfig.Name, c.runner.GetInstancePid(localInstanceConfig.Name))
	}
}

func (c *Client) Stop() (err error) {
	return c.runner.Close()
}

func (c *Client) StartFrpInstance(instance types.InstanceConfigLocal) (err error) {
	var frpPath string
	frpPath, err = installer.IsFRPInstalled(c.binDir, instance.Version)
	if err != nil {
		log.Printf("FRP版本 %s 未安装，开始安装", instance.Version)
		frpPath, err = installer.EnsureFRPInstalled(c.binDir, instance.Version)
		if err != nil {
			log.Fatalf("安装FRP版本 %s 失败: %v", instance.Version, err)
		}
		log.Printf("FRP版本 %s 安装成功", instance.Version)
	}
	if err := c.runner.StartInstance(instance.Name, instance.Version, frpPath, instance.ConfigPath); err != nil {
		log.Printf("启动实例 %s 失败: %v", instance.Name, err)
	}
	return
}

func (c *Client) StopFrpInstance(name string) (err error) {
	if err := c.runner.StopInstance(name); err != nil {
		log.Printf("停止实例 %s 失败: %v", name, err)
	}
	return
}
