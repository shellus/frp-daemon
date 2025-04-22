package fdclient

import (
	"errors"
	"os"

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
	// 如果文件不存在，调用WriteInstancesFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = WriteInstancesFile(path, &InstancesFile{})
		if err != nil {
			return nil, err
		}
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

// WriteInstancesFile 写入instances.yaml配置
func WriteInstancesFile(path string, config *InstancesFile) error {
	if path == "" {
		return errors.New("path is empty")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
