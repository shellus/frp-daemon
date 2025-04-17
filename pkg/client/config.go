package config

import (
	"errors"
	"os"

	"github.com/shellus/frp-daemon/pkg/types"
	"gopkg.in/yaml.v3"
)

// ClientConfig client.yaml配置
type ClientConfig struct {
	Client types.ClientAuth `json:"client"` // 客户端认证信息
	Mqtt   types.MQTTConfig `json:"mqtt"`   // MQTT配置
}

// InstanceConfig FRP实例配置
type InstanceConfig struct {
	Name    string `json:"name"`    // 实例名称
	Version string `json:"version"` // FRP版本
	ConfigPath  string `json:"configPath"`  // FRP配置文件
}

// InstancesFile instances.yaml配置
type InstancesFile struct {
	Instances []InstanceConfig `json:"instances"` // FRP实例配置
}

// LoadClientConfig 加载守护进程配置
func LoadClientConfig(path string) (*ClientConfig, error) {
	if path == "" {
		return nil, errors.New("path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ClientConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadInstancesFile 加载instances.yaml配置
func LoadInstancesFile(path string) (*InstancesFile, error) {
	if path == "" {
		return nil, errors.New("path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config InstancesFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
