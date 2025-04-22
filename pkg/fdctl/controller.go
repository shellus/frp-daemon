package fdctl

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/shellus/frp-daemon/pkg/mqtt"
	"github.com/shellus/frp-daemon/pkg/types"
)

type Controller struct {
	auth       types.ClientAuth
	mqttClient *mqtt.MQTT
	mqttOpts   types.MQTTClientOpts
}

func NewController(auth types.ClientAuth, mqttOpts types.MQTTClientOpts) (*Controller, error) {
	if auth.ClientId == "" {
		return nil, errors.New("auth.ClientId is empty")
	}
	if mqttOpts.Broker == "" {
		return nil, errors.New("mqttOpts.Broker is empty")
	}
	return &Controller{
		auth:     auth,
		mqttOpts: mqttOpts,
	}, nil
}

// 连接MQTT
func (c *Controller) ConnectMQTT() error {
	mqttClient, err := mqtt.NewMQTT(c.mqttOpts)
	if err != nil {
		return fmt.Errorf("mqtt connect failed: %v", err)
	}
	if err := mqttClient.Connect(); err != nil {
		return fmt.Errorf("mqtt connect failed: %v", err)
	}

	c.mqttClient = mqttClient
	return nil
}

// 实现配置下发
func (c *Controller) SendConfig(clientId string, config *types.InstanceConfigLocal) error {
	if clientId == "" {
		return errors.New("clientId is empty")
	}
	if config == nil {
		return errors.New("config is nil")
	}

	// 构建MQTT主题
	topic := mqtt.MessageTopic(c.mqttOpts.TopicPrefix, clientId)

	// 读取配置文件内容
	configContent, err := os.ReadFile(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("read config file failed: %v", err)
	}

	// 创建远程配置对象
	remoteConfig := types.InstanceConfigRemote{
		Name:          config.Name,
		Version:       config.Version,
		ConfigContent: string(configContent),
	}

	// 序列化配置
	configJSON, err := json.Marshal(remoteConfig)
	if err != nil {
		return fmt.Errorf("marshal config failed: %v", err)
	}

	// 创建消息
	message := types.Message{
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		MessageId:        types.GenerateRandomString(16),
		Type:             types.Req,
		Action:           types.MessageActionUpdate,
		Payload:          configJSON,
	}

	// 发布消息
	if err := c.mqttClient.Publish(topic, message, byte(c.mqttOpts.QoS), c.mqttOpts.Retain); err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}

	return nil
}

// SendPing 发送ping消息到指定客户端
func (c *Controller) SendPing(clientId string) error {
	if clientId == "" {
		return errors.New("clientId is empty")
	}

	// 构建MQTT主题
	topic := mqtt.MessageTopic(c.mqttOpts.TopicPrefix, clientId)

	// 创建消息
	message := types.Message{
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		MessageId:        types.GenerateRandomString(16),
		Type:             types.Req,
		Action:           types.MessageActionPing,
		Payload:          nil,
	}

	// 发布消息
	if err := c.mqttClient.Publish(topic, message, byte(c.mqttOpts.QoS), c.mqttOpts.Retain); err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}

	return nil
}

// 列出客户端实例
func (c *Controller) ListInstances(clientId string) ([]types.InstanceConfigLocal, error) {

	return nil, errors.New("not implemented")
}

// 删除客户端实例
func (c *Controller) DeleteInstance(clientId string, instanceName string) error {

	return errors.New("not implemented")
}

// 查看指定实例的lastLog
func (c *Controller) GetLastLog(clientId string, instanceName string) ([]string, error) {

	return nil, errors.New("not implemented")
}

// 查看指定实例的status
func (c *Controller) GetStatus(clientId string, instanceName string) ([]types.InstanceStatus, error) {

	return nil, errors.New("not implemented")
}

//
