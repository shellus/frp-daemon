package fdctl

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/shellus/frp-daemon/pkg/mqtt"
	"github.com/shellus/frp-daemon/pkg/mqtt/task"
	"github.com/shellus/frp-daemon/pkg/types"
)

type Controller struct {
	auth       types.ClientAuth
	MqttClient *mqtt.MQTT
	mqttOpts   types.MQTTClientOpts
	logger     zerolog.Logger
}

func NewController(auth types.ClientAuth, mqttOpts types.MQTTClientOpts, logger zerolog.Logger) (*Controller, error) {
	if auth.ClientId == "" {
		return nil, errors.New("auth.ClientId is empty")
	}
	if mqttOpts.Broker == "" {
		return nil, errors.New("mqttOpts.Broker is empty")
	}
	return &Controller{
		auth:     auth,
		mqttOpts: mqttOpts,
		logger:   logger,
	}, nil
}

// 连接MQTT
func (c *Controller) ConnectMQTT() error {
	mqttClient, err := mqtt.NewMQTT(c.mqttOpts, c.logger)
	if err != nil {
		return fmt.Errorf("mqtt connect failed: %v", err)
	}
	if err := mqttClient.Connect(); err != nil {
		return fmt.Errorf("mqtt connect failed: %v", err)
	}

	c.MqttClient = mqttClient
	return nil
}

// 实现配置下发
func (c *Controller) SendConfig(clientId string, clientPassword string, config types.InstanceConfigLocal) error {
	if clientId == "" {
		return errors.New("下发配置要发送到的clientId为空")
	}

	// 读取配置文件内容
	configContent, err := os.ReadFile(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("下发配置读取frpc.ini文件失败，err=%v，configPaht=%s", err, config.ConfigPath)
	}

	// 创建远程配置对象
	remoteConfig := types.InstanceConfigRemote{
		ClientPassword: clientPassword,
		Name:           config.Name,
		Version:        config.Version,
		ConfigContent:  string(configContent),
	}

	// 序列化配置
	configJSON, err := json.Marshal(remoteConfig)
	if err != nil {
		return err
	}
	// 异步行为调用
	err = c.MqttClient.RsyncAction(task.MessagePending{
		MessageId:        types.GenerateRandomString(16),
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		Action:           types.MessageActionUpdate,
		Payload:          configJSON,
		Expiration:       time.Now().Add(3 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		return fmt.Errorf("下发配置发送失败，err=%v", err)
	}
	c.logger.Info().Msg("下发配置发送成功")
	return nil
}

// SendPing 发送ping消息到指定客户端
func (c *Controller) SendPing(clientId string) error {
	if clientId == "" {
		return errors.New("clientId is empty")
	}
	pingMessage := types.PingMessage{
		Time: time.Now().UnixMilli(),
	}
	pingMessageJSON, err := json.Marshal(pingMessage)
	if err != nil {
		return fmt.Errorf("marshal ping message failed: %v", err)
	}

	// 同步行为调用
	waiter, err := c.MqttClient.SyncAction(task.MessagePending{
		MessageId:        types.GenerateRandomString(16),
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		Action:           types.MessageActionPing,
		Payload:          json.RawMessage(pingMessageJSON),
		Expiration:       time.Now().Add(10 * time.Second).Unix(),
	})
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}

	remoteResult, err := waiter.Wait()
	if err != nil {
		return fmt.Errorf("延迟测试远端执行失败，err=%v", err)
	}
	if remoteResult == nil {
		return errors.New("延迟测试远端执行失败，value为空")
	}
	// json反序列化
	var pingResult types.PingMessage
	if err := json.Unmarshal(remoteResult, &pingResult); err != nil {
		return fmt.Errorf("延迟测试远端结果反序列化失败，err=%v", err)
	}
	c.logger.Info().Msgf("延迟测试远端结果，单向延迟=%d，双向延迟=%d", pingResult.Time-pingMessage.Time, time.Now().UnixMilli()-pingMessage.Time)
	return nil
}

// 列出客户端实例
func (c *Controller) ListInstances(clientId string) ([]types.InstanceConfigLocal, error) {
	return nil, errors.New("not implemented")
}

// 删除客户端实例
func (c *Controller) DeleteInstance(clientId string, instanceName string) error {
	if clientId == "" {
		return errors.New("clientId is empty")
	}
	if instanceName == "" {
		return errors.New("instanceName is empty")
	}

	// 创建删除实例的消息
	deleteMessage := types.DeleteInstanceMessage{
		InstanceName: instanceName,
	}
	deleteMessageJSON, err := json.Marshal(deleteMessage)
	if err != nil {
		return fmt.Errorf("marshal delete message failed: %v", err)
	}

	// 同步行为调用
	waiter, err := c.MqttClient.SyncAction(task.MessagePending{
		MessageId:        types.GenerateRandomString(16),
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		Action:           types.MessageActionDelete,
		Payload:          json.RawMessage(deleteMessageJSON),
		Expiration:       time.Now().Add(10 * time.Second).Unix(),
	})
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}

	remoteResult, err := waiter.Wait()
	if err != nil {
		return fmt.Errorf("删除实例远端执行失败，err=%v", err)
	}
	if remoteResult == nil {
		return errors.New("删除实例远端执行失败，value为空")
	}

	c.logger.Info().Msgf("删除实例成功，clientId=%s, instanceName=%s", clientId, instanceName)
	return nil
}

// 查看指定实例的lastLog
func (c *Controller) GetLastLog(clientId string, instanceName string) ([]string, error) {

	return nil, errors.New("not implemented")
}

// 查看指定实例的status
func (c *Controller) GetStatus(clientId string, instanceName string) (*types.InstanceStatus, error) {
	if clientId == "" {
		return nil, errors.New("clientId is empty")
	}
	if instanceName == "" {
		return nil, errors.New("instanceName is empty")
	}

	// 创建获取状态的消息
	statusMessage := types.GetStatusMessage{
		InstanceName: instanceName,
	}
	statusMessageJSON, err := json.Marshal(statusMessage)
	if err != nil {
		return nil, fmt.Errorf("marshal status message failed: %v", err)
	}

	// 同步行为调用
	waiter, err := c.MqttClient.SyncAction(task.MessagePending{
		MessageId:        types.GenerateRandomString(16),
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		Action:           types.MessageActionGetStatus,
		Payload:          json.RawMessage(statusMessageJSON),
		Expiration:       time.Now().Add(10 * time.Second).Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("publish failed: %v", err)
	}

	remoteResult, err := waiter.Wait()
	if err != nil {
		return nil, fmt.Errorf("获取状态远端执行失败，err=%v", err)
	}
	if remoteResult == nil {
		return nil, errors.New("获取状态远端执行失败，value为空")
	}

	// 解析状态
	var status types.InstanceStatus
	if err := json.Unmarshal(remoteResult, &status); err != nil {
		return nil, fmt.Errorf("解析状态失败，err=%v", err)
	}

	return &status, nil
}

// SendWOL 发送WOL命令到指定客户端
func (c *Controller) SendWOL(clientId string, macAddress string) error {
	if clientId == "" {
		return errors.New("clientId is empty")
	}
	if macAddress == "" {
		return errors.New("macAddress is empty")
	}

	// 创建WOL消息
	wolMessage := types.WOLMessage{
		MacAddress: macAddress,
	}
	wolMessageJSON, err := json.Marshal(wolMessage)
	if err != nil {
		return fmt.Errorf("marshal wol message failed: %v", err)
	}

	// 同步行为调用
	waiter, err := c.MqttClient.SyncAction(task.MessagePending{
		MessageId:        types.GenerateRandomString(16),
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		Action:           types.MessageActionWOL,
		Payload:          json.RawMessage(wolMessageJSON),
		Expiration:       time.Now().Add(10 * time.Second).Unix(),
	})
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}

	remoteResult, err := waiter.Wait()
	if err != nil {
		return fmt.Errorf("发送WOL命令远端执行失败，err=%v", err)
	}
	if remoteResult == nil {
		return errors.New("发送WOL命令远端执行失败，value为空")
	}

	c.logger.Info().Msgf("发送WOL命令成功，clientId=%s, macAddress=%s", clientId, macAddress)
	return nil
}

// SendShutdownWindows 发送Windows远程关机消息到指定客户端
func (c *Controller) SendShutdownWindows(clientId string, ip, username, password string) error {
	if clientId == "" {
		return errors.New("clientId is empty")
	}
	if ip == "" {
		return errors.New("ip is empty")
	}
	if username == "" {
		return errors.New("username is empty")
	}
	if password == "" {
		return errors.New("password is empty")
	}

	// 创建Windows远程关机消息
	shutdownMessage := types.ShutdownWindowsMessage{
		IP:       ip,
		Username: username,
		Password: password,
	}
	shutdownMessageJSON, err := json.Marshal(shutdownMessage)
	if err != nil {
		return fmt.Errorf("marshal shutdown message failed: %v", err)
	}

	// 同步行为调用
	waiter, err := c.MqttClient.SyncAction(task.MessagePending{
		MessageId:        types.GenerateRandomString(16),
		SenderClientId:   c.auth.ClientId,
		ReceiverClientId: clientId,
		Action:           types.MessageActionShutdownWindows,
		Payload:          json.RawMessage(shutdownMessageJSON),
		Expiration:       time.Now().Add(10 * time.Second).Unix(),
	})
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}

	remoteResult, err := waiter.Wait()
	if err != nil {
		return fmt.Errorf("Windows远程关机远端执行失败，err=%v", err)
	}
	if remoteResult == nil {
		return errors.New("Windows远程关机远端执行失败，value为空")
	}

	c.logger.Info().Msgf("Windows远程关机成功，clientId=%s, ip=%s", clientId, ip)
	return nil
}

//
