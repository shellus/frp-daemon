package fdclient

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/shellus/frp-daemon/pkg/types"
	"gopkg.in/yaml.v3"
)

// ClientConfig client.yaml配置
type ClientConfig struct {
	Client    types.ClientAuth            `yaml:"client"`    // 客户端认证信息
	Mqtt      types.MQTTClientOpts        `yaml:"mqtt"`      // MQTT连接配置
	Instances []types.InstanceConfigLocal `yaml:"instances"` // FRP实例配置
}
type ConfigFile struct {
	path         string
	ClientConfig ClientConfig `yaml:"client"` // 客户端配置
	instMutex    sync.Mutex   // 实例配置互斥锁，当添加删除更新实例配置时，需要加锁
}

// LoadClientConfig 加载守护进程配置
func LoadClientConfig(path string) (*ConfigFile, error) {
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

	return &ConfigFile{
		path:         path,
		ClientConfig: config,
	}, nil
}

// WriteInstancesFile 追加instances配置项目
func (cf *ConfigFile) AddInstance(config types.InstanceConfigLocal) error {
	cf.ClientConfig.Instances = append(cf.ClientConfig.Instances, config)
	data, err := yaml.Marshal(cf.ClientConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(cf.path, data, 0644)
}

func (cf *ConfigFile) save() error {
	data, err := yaml.Marshal(cf.ClientConfig)
	if err != nil {
		return err
	}
	return os.WriteFile(cf.path, data, 0644)
}

func (cf *ConfigFile) indexInstance(name string) (int, error) {
	for i, instance := range cf.ClientConfig.Instances {
		if instance.Name == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("实例配置不存在，name=%s", name)
}

// RemoveInstance 删除instances配置项目
func (cf *ConfigFile) RemoveInstance(name string) error {
	cf.instMutex.Lock()
	defer cf.instMutex.Unlock()
	index, err := cf.indexInstance(name)
	if err != nil {
		return err
	}
	cf.ClientConfig.Instances = append(cf.ClientConfig.Instances[:index], cf.ClientConfig.Instances[index+1:]...)

	return cf.save()
}

// UpdateInstance 更新instances配置项目，新增或替换
func (cf *ConfigFile) UpdateInstance(config types.InstanceConfigLocal) error {
	cf.instMutex.Lock()
	defer cf.instMutex.Unlock()
	index, err := cf.indexInstance(config.Name)
	if err != nil {
		cf.ClientConfig.Instances = append(cf.ClientConfig.Instances, config)
	} else {
		cf.ClientConfig.Instances[index] = config
	}

	return cf.save()
}

// GetInstance 获取instances配置项目
func (cf *ConfigFile) GetInstance(name string) (types.InstanceConfigLocal, error) {
	cf.instMutex.Lock()
	defer cf.instMutex.Unlock()
	index, err := cf.indexInstance(name)
	if err != nil {
		return types.InstanceConfigLocal{}, err
	}
	return cf.ClientConfig.Instances[index], nil
}
