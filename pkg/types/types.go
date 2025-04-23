package types

import (
	"crypto/rand"
	"encoding/hex"
)

const (
	TopicPrefix = "frp-client"
)

// MQTTClientOpts MQTT客户端选项
type MQTTClientOpts struct {
	Broker      string `yaml:"broker"`       // MQTT代理地址
	ClientID    string `yaml:"client_id"`    // MQTT客户端ID
	Username    string `yaml:"username"`     // MQTT用户名
	Password    string `yaml:"password"`     // MQTT密码
	TopicPrefix string `yaml:"topic_prefix"` // MQTT主题前缀
}

// ClientConfig 客户端配置，这是本程序的客户端配置，不是MQTT的客户端配置
type ClientAuth struct {
	Name     string `yaml:"name"`      // 客户端名称，无实际用途
	ClientId string `yaml:"client_id"` // 客户端ID，会作为mqtt的用户名
	Password string `yaml:"password"`  // 客户端密码，会作为mqtt的密码
}

// generateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// InstanceConfigLocal FRP实例配置-本地
type InstanceConfigLocal struct {
	Name       string `yaml:"name"`       // 实例名称
	Version    string `yaml:"version"`    // FRP版本
	ConfigPath string `yaml:"configPath"` // FRP配置文件
}

// InstanceConfigRemote FRP实例配置-远程
type InstanceConfigRemote struct {
	ClientPassword string `yaml:"client_password"` // 客户端密码，要进行远程配置下发必须验证密码
	Name           string `yaml:"name"`            // 实例名称
	Version        string `yaml:"version"`         // FRP版本
	ConfigContent  string `yaml:"config_content"`  // FRP配置文件内容
}

// Status 客户端状态
type Status struct {
	ID        string           `json:"id"`        // 客户端ID
	Online    bool             `json:"online"`    // 是否在线
	Instances []InstanceStatus `json:"instances"` // 实例状态
}

// InstanceStatus FRP实例状态
type InstanceStatus struct {
	Running    bool     `json:"running"`     // 是否运行中
	LastLog    []string `json:"last_log"`    // 最后100行日志
	ExitStatus int      `json:"exit_status"` // 退出状态
	Pid        int      `json:"pid"`         // 进程ID
}

// EMQXAPIConfig EMQX API配置，控制端用来创建MQTT用户使用
type EMQXAPIConfig struct {
	ApiEndpoint  string `yaml:"api_endpoint"`   // API端点
	ApiAppKey    string `yaml:"api_app_key"`    // API App Key
	ApiSecretKey string `yaml:"api_secret_key"` // API Secret Key
	MQTTBroker   string `yaml:"mqtt_broker"`    // MQTT Broker
}

const (
	// MessageActionPing 心跳
	MessageActionPing string = "ping"
	// MessageActionUpdate 对应的Payload是InstanceConfigRemote
	MessageActionUpdate string = "update"
)

type PingMessage struct {
	Time int64 `json:"time"` // 时间戳，毫秒
}
