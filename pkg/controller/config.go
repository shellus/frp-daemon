package controller

import (
	"os"

	"github.com/shellus/frp-daemon/pkg/types"
	"gopkg.in/yaml.v3"
)

// ControllerConfig 控制器配置
type ControllerConfig struct {
	EMQXAPI *types.EMQXAPIConfig  `yaml:"emqx_api"` // EMQX API配置
	Client  *types.ClientAuth     `yaml:"client"`   // 客户端认证信息，控制端也是一个客户端，所以也有自己的客户端配置
	MQTT    *types.MQTTClientOpts `yaml:"mqtt"`     // MQTT连接配置，用于发送控制指令到MQTT
	Clients map[string]*types.ClientAuth `yaml:"clients"` // 被控端客户端列表
}

// LoadControllerConfig 加载控制器配置
func LoadControllerConfig(configPath string) (*ControllerConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ControllerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
