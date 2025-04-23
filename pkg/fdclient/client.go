package fdclient

import (
	"fmt"
	"log"
	"os"

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
}

func NewClient(configFile *ConfigFile, runner *frp.Runner, binDir, instancesDir string) (*Client, error) {
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
	installer, err := installerC.NewInstaller(binDir, "")
	if err != nil {
		return nil, fmt.Errorf("创建安装器失败，Error=%v", err)
	}

	c := &Client{
		configFile:   configFile,
		runner:       runner,
		binDir:       binDir,
		instancesDir: instancesDir,
		installer:    installer,
	}

	mqtt, err := mqttC.NewMQTT(configFile.ClientConfig.Mqtt)
	if err != nil {
		return nil, err
	}

	mqtt.SubscribeAction(types.MessageActionUpdate, c.HandleUpdate)
	mqtt.SubscribeAction(types.MessageActionPing, c.HandlePing)

	if err := mqtt.Connect(); err != nil {
		return nil, err
	}

	c.mqtt = mqtt

	return c, nil
}

func (c *Client) Start() (err error) {
	for _, localInstanceConfig := range c.configFile.ClientConfig.Instances {
		if err := c.StartFrpInstance(localInstanceConfig); err != nil {
			log.Printf("启动实例失败但继续，InstanceName=%s, Error=%v", localInstanceConfig.Name, err)
		}
		log.Printf("启动实例成功，InstanceName=%s, Pid=%d", localInstanceConfig.Name, c.runner.GetInstancePid(localInstanceConfig.Name))
	}
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
