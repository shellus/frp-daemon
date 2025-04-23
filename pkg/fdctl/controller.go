package fdctl

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shellus/frp-daemon/pkg/mqtt"
	"github.com/shellus/frp-daemon/pkg/mqtt/task"
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
		return errors.New("要发送到的clientId为空")
	}
	if config == nil {
		return errors.New("实例config为空")
	}

	// 读取配置文件内容
	configContent, err := os.ReadFile(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("读取frpc.ini文件失败，err=%v，configPaht=%s", err, config.ConfigPath)
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
		return err
	}

	// 发布消息 TODO 改为task.Message的封装，业务层只操作任务，不直接操作mqtt的发布订阅。
	waiter, err := c.mqttClient.SyncAction(task.MessagePending{
		MessageId:        types.GenerateRandomString(16),
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		Action:           types.MessageActionUpdate,
		Payload:          configJSON,
	})
	if err != nil {
		return fmt.Errorf("同步调用行为失败，err=%v", err)
	}
	remoteResult, err := waiter.Wait()
	if err != nil {
		return fmt.Errorf("同步调用行为远端执行失败，err=%v", err)
	}
	if remoteResult == nil {
		return errors.New("同步调用行为远端执行失败，value为空")
	}
	log.Printf("同步调用行为远端结果，value=%v", remoteResult)
	return nil
}

// SendPing 发送ping消息到指定客户端
func (c *Controller) SendPing(clientId string) error {
	if clientId == "" {
		return errors.New("clientId is empty")
	}
	pingMessage := types.PingMessage{
		Time: time.Now().Unix(),
	}
	pingMessageJSON, err := json.Marshal(pingMessage)
	if err != nil {
		return fmt.Errorf("marshal ping message failed: %v", err)
	}
	// 创建消息
	message := task.MessagePending{
		MessageId:        types.GenerateRandomString(16),
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		Action:           types.MessageActionPing,
		Payload:          json.RawMessage(pingMessageJSON),
	}

	// 发布消息
	waiter, err := c.mqttClient.SyncAction(message)
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}

	remoteResult, err := waiter.Wait()
	if err != nil {
		return fmt.Errorf("同步调用行为远端执行失败，err=%v", err)
	}
	if remoteResult == nil {
		return errors.New("同步调用行为远端执行失败，value为空")
	}
	log.Printf("同步调用行为远端结果，value=%v", remoteResult)
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
