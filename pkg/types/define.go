package types

const (
	TopicPrefix = "frp-client"
)

// InstanceConfigLocal FRP实例配置-本地
type InstanceConfigLocal struct {
	Name       string `yaml:"name"`       // 实例名称
	Version    string `yaml:"version"`    // FRP版本
	ConfigPath string `yaml:"configPath"` // FRP配置文件
}

// EMQXAPIConfig EMQX API配置，控制端用来创建MQTT用户使用
type EMQXAPIConfig struct {
	ApiEndpoint  string `yaml:"api_endpoint"`   // API端点
	ApiAppKey    string `yaml:"api_app_key"`    // API App Key
	ApiSecretKey string `yaml:"api_secret_key"` // API Secret Key
	MQTTBroker   string `yaml:"mqtt_broker"`    // MQTT Broker
}

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
