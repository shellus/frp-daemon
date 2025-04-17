package types

// MQTTConfig MQTT连接配置
type MQTTConfig struct {
	Broker   string `json:"broker"`    // MQTT服务器地址
	Username string `json:"username"`  // 用户名
	Password string `json:"password"`  // 密码
	ClientID string `json:"client_id"` // 客户端ID
}

// ClientAuth 客户端认证信息
type ClientAuth struct {
	ID       string `json:"id"`       // 客户端ID
	Password string `json:"password"` // 认证密码
}


// Status 客户端状态
type Status struct {
	ID        string           `json:"id"`        // 客户端ID
	Online    bool             `json:"online"`    // 是否在线
	Instances []InstanceStatus `json:"instances"` // 实例状态
}

// InstanceStatus FRP实例状态
type InstanceStatus struct {
	Running bool   `json:"running"` // 是否运行中
	LastLog string `json:"last_log"` // 最后100行日志
	ExitStatus int `json:"exit_status"` // 退出状态
}
