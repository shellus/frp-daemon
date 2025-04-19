package types

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
)

const (
	// MQTT的默认配置
	TopicPrefix  = "frp-client"
	QoS          = 1
	Retain       = true
	CleanSession = true
)

// MQTTClientOpts MQTT客户端选项
type MQTTClientOpts struct {
	Broker       string `yaml:"broker"`        // MQTT代理地址
	ClientID     string `yaml:"client_id"`     // MQTT客户端ID
	Username     string `yaml:"username"`      // MQTT用户名
	Password     string `yaml:"password"`      // MQTT密码
	TopicPrefix  string `yaml:"topic_prefix"`  // MQTT主题前缀
	QoS          int    `yaml:"qos"`           // MQTT QoS
	Retain       bool   `yaml:"retain"`        // MQTT保留消息
	CleanSession bool   `yaml:"clean_session"` // MQTT清理会话
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
	Name          string `yaml:"name"`           // 实例名称
	Version       string `yaml:"version"`        // FRP版本
	ConfigContent string `yaml:"config_content"` // FRP配置文件内容
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

// MessageAction 消息动作常量
type MessageAction string

const (
	// MessageActionPing 心跳
	MessageActionPing MessageAction = "ping"
	// MessageActionUpdate 对应的Payload是InstanceConfigRemote
	MessageActionUpdate MessageAction = "update"
)

type MessageType string

const (
	// Req 请求
	Req MessageType = "req"
	// Resp 响应
	Resp MessageType = "resp"
)

// Message MQTT共用消息外包装，双向数据体都是这个
type Message struct {
	SenderClientId   string          `json:"sender_client_id"`   // 发送者客户端ID
	ReceiverClientId string          `json:"receiver_client_id"` // 接收者客户端ID
	MessageId        string          `json:"message_id"`         // 消息ID，一般为UUID，resp和req的message_id相同
	Type             MessageType     `json:"type"`               // 消息类型
	Action           MessageAction   `json:"action"`             // 消息动作
	Payload          json.RawMessage `json:"payload"`            // 消息负载
}

type PingMessage struct {
	Time int64 `json:"time"` // 时间戳
}
