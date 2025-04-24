package fdclient

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/shellus/frp-daemon/pkg/frp"
	installerC "github.com/shellus/frp-daemon/pkg/installer"
	mqttC "github.com/shellus/frp-daemon/pkg/mqtt"
	"github.com/shellus/frp-daemon/pkg/types"
)

type Client struct {
	configFile   *ConfigFile
	mqtt         *mqttC.MQTT
	runner       *frp.Runner
	binDir       string
	instancesDir string
	installer    *installerC.Installer
	logger       zerolog.Logger
}

func NewClient(configFile *ConfigFile, runner *frp.Runner, binDir, instancesDir string, logger zerolog.Logger) (*Client, error) {
	if configFile.ClientConfig.Client.ClientId == "" {
		return nil, fmt.Errorf("配置错误，auth.ClientId is empty")
	}
	if configFile.ClientConfig.Mqtt.Broker == "" {
		return nil, fmt.Errorf("配置错误，mqtt.Broker is empty")
	}
	if _, err := os.Stat(instancesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("FRP实例目录不存在，instancesDir=%s", instancesDir)
	}
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("FRP二进制目录不存在，binDir=%s", binDir)
	}
	installer, err := installerC.NewInstaller(binDir, "", logger)
	if err != nil {
		return nil, fmt.Errorf("创建安装器失败，Error=%v", err)
	}

	c := &Client{
		configFile:   configFile,
		runner:       runner,
		binDir:       binDir,
		instancesDir: instancesDir,
		installer:    installer,
		logger:       logger,
	}

	mqtt, err := mqttC.NewMQTT(configFile.ClientConfig.Mqtt, logger)
	if err != nil {
		return nil, fmt.Errorf("创建MQTT客户端失败，Error=%v", err)
	}

	mqtt.SubscribeAction(types.MessageActionUpdate, c.HandleUpdate)
	mqtt.SubscribeAction(types.MessageActionPing, c.HandlePing)
	mqtt.SubscribeAction(types.MessageActionDelete, c.HandleDelete)
	mqtt.SubscribeAction(types.MessageActionGetStatus, c.HandleGetStatus)
	mqtt.SubscribeAction(types.MessageActionWOL, c.HandleWOL)

	if err := mqtt.Connect(); err != nil {
		return nil, fmt.Errorf("连接MQTT失败，Error=%v", err)
	}

	c.mqtt = mqtt

	return c, nil
}

func (c *Client) Start() (err error) {
	for _, localInstanceConfig := range c.configFile.ClientConfig.Instances {
		if err := c.StartFrpInstance(localInstanceConfig); err != nil {
			c.logger.Warn().Msgf("启动实例失败但继续，InstanceName=%s, Error=%v", localInstanceConfig.Name, err)
		}
		c.logger.Info().Msgf("启动实例成功，InstanceName=%s, Pid=%d", localInstanceConfig.Name, c.runner.GetInstancePid(localInstanceConfig.Name))
	}
	return
}

// ReportStatus 上报状态，应该被每分钟调用一次
func (c *Client) ReportStatus() (err error) {
	instancesStatus := c.runner.GetStatus()
	status := types.Status{
		ID:             c.configFile.ClientConfig.Client.ClientId,
		LastOnlineTime: time.Now().Unix(),
		Instances:      instancesStatus,
	}

	c.mqtt.Report(c.configFile.ClientConfig.Client.ClientId, status)

	return
}

func (c *Client) Stop() (err error) {
	return c.runner.Close()
}

func (c *Client) StartFrpInstance(instance types.InstanceConfigLocal) (err error) {
	frpPath, err := c.installer.EnsureFRPInstalled(instance.Version)
	if err != nil {
		return err
	}
	return c.runner.StartInstance(instance.Name, instance.Version, frpPath, instance.ConfigPath)
}

func (c *Client) StopFrpInstance(name string) (err error) {
	return c.runner.StopInstance(name)
}

// HandleDelete 处理删除实例
func (c *Client) HandleDelete(action string, payload []byte) (value []byte, err error) {
	var deleteMessage types.DeleteInstanceMessage
	if err = json.Unmarshal(payload, &deleteMessage); err != nil {
		return nil, fmt.Errorf("处理delete指令解析失败，Error=%v", err)
	}
	c.logger.Info().Msgf("处理delete指令，instanceName=%s", deleteMessage.InstanceName)

	// 停止实例
	if err = c.StopFrpInstance(deleteMessage.InstanceName); err != nil {
		c.logger.Error().Msgf("停止实例失败，instanceName=%s, Error=%v", deleteMessage.InstanceName, err)
		return nil, fmt.Errorf("停止实例失败，instanceName=%s, Error=%v", deleteMessage.InstanceName, err)
	}

	// 删除实例配置
	if err = c.configFile.RemoveInstance(deleteMessage.InstanceName); err != nil {
		c.logger.Error().Msgf("删除实例配置失败，instanceName=%s, Error=%v", deleteMessage.InstanceName, err)
		return nil, fmt.Errorf("删除实例配置失败，instanceName=%s, Error=%v", deleteMessage.InstanceName, err)
	}

	c.logger.Info().Msgf("处理delete指令完成，instanceName=%s", deleteMessage.InstanceName)

	respByte, err := json.Marshal("搞完了")
	if err != nil {
		return nil, fmt.Errorf("序列化响应失败，Error=%v", err)
	}
	return respByte, nil
}

// HandleGetStatus 处理获取状态
func (c *Client) HandleGetStatus(action string, payload []byte) (value []byte, err error) {
	var statusMessage types.GetStatusMessage
	if err = json.Unmarshal(payload, &statusMessage); err != nil {
		return nil, fmt.Errorf("处理get_status指令解析失败，Error=%v", err)
	}
	c.logger.Info().Msgf("处理get_status指令，instanceName=%s", statusMessage.InstanceName)

	// 获取状态
	status := c.runner.GetStatus()

	// 查找指定实例的状态
	var instanceStatus *types.InstanceStatus
	for _, s := range status {
		if s.Name == statusMessage.InstanceName {
			instanceStatus = &s
			break
		}
	}

	if instanceStatus == nil {
		return nil, fmt.Errorf("未找到实例 %s 的状态", statusMessage.InstanceName)
	}

	// 序列化状态
	statusJSON, err := json.Marshal(instanceStatus)
	if err != nil {
		return nil, fmt.Errorf("序列化状态失败，Error=%v", err)
	}

	return statusJSON, nil
}

// HandleWOL 处理WOL消息
func (c *Client) HandleWOL(action string, payload []byte) (value []byte, err error) {
	var wolMessage types.WOLMessage
	if err = json.Unmarshal(payload, &wolMessage); err != nil {
		return nil, fmt.Errorf("处理wol指令解析失败，Error=%v", err)
	}
	c.logger.Info().Msgf("处理wol指令，macAddress=%s", wolMessage.MacAddress)

	// 发送WOL包
	if err = c.sendWOLPacket(wolMessage.MacAddress); err != nil {
		return nil, fmt.Errorf("发送WOL包失败，Error=%v", err)
	}

	respByte, err := json.Marshal("WOL包发送成功")
	if err != nil {
		return nil, fmt.Errorf("序列化响应失败，Error=%v", err)
	}
	return respByte, nil
}

// sendWOLPacket 发送WOL包
func (c *Client) sendWOLPacket(macAddress string) error {
	// 解析MAC地址
	mac, err := net.ParseMAC(macAddress)
	if err != nil {
		return fmt.Errorf("解析MAC地址失败，Error=%v", err)
	}

	// 创建WOL包
	packet := make([]byte, 102)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 6; i < 102; i += 6 {
		copy(packet[i:], mac)
	}

	// 发送WOL包
	addr := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 9,
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("创建UDP连接失败，Error=%v", err)
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("发送WOL包失败，Error=%v", err)
	}

	return nil
}
