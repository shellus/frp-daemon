package client

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/shellus/frp-daemon/pkg/types"
	"gopkg.in/yaml.v3"
)

// ClientConfig client.yaml配置
type ClientConfig struct {
	Client types.ClientAuth     `yaml:"client"` // 客户端认证信息
	Mqtt   types.MQTTClientOpts `yaml:"mqtt"`   // MQTT连接配置
}

// InstancesFile instances.yaml配置
type InstancesFile struct {
	Instances []types.InstanceConfigLocal `yaml:"instances"` // FRP实例配置
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

	// 获取instances.yaml的绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	var config InstancesFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
