package types

// 传输的消息体

// Status 客户端状态，仅被控端向控制端回复
type Status struct {
	ID             string           `json:"id"`               // 客户端ID
	LastOnlineTime int64            `json:"last_online_time"` // 最后在线时间
	Instances      []InstanceStatus `json:"instances"`        // 实例状态
}

// InstanceStatus FRP实例状态，仅被控端向控制端回复
type InstanceStatus struct {
	Name       string   `json:"name"`        // 实例名称
	Running    bool     `json:"running"`     // 是否运行中
	StartTime  int64    `json:"start_time"`  // 启动时间, 单位为秒
	ExitTime   int64    `json:"exit_time"`   // 退出时间, 单位为秒
	LastLog    []string `json:"last_log"`    // 最后100行日志
	ExitStatus int      `json:"exit_status"` // 退出状态
	Pid        int      `json:"pid"`         // 进程ID
}

// PingMessage 心跳消息，双向共用
type PingMessage struct {
	Time int64 `json:"time"` // 时间戳，毫秒
}

// InstanceConfigRemote FRP实例配置-远程，仅控制端向被控端下发
type InstanceConfigRemote struct {
	ClientPassword string `yaml:"client_password"` // 客户端密码，要进行远程配置下发必须验证密码
	Name           string `yaml:"name"`            // 实例名称
	Version        string `yaml:"version"`         // FRP版本
	ConfigContent  string `yaml:"config_content"`  // FRP配置文件内容
}

// DeleteInstanceMessage 删除实例消息，仅控制端向被控端下发
type DeleteInstanceMessage struct {
	InstanceName string `json:"instance_name"` // 实例名称
}

// GetStatusMessage 获取状态消息，仅控制端向被控端下发
type GetStatusMessage struct {
	InstanceName string `json:"instance_name"` // 实例名称
}

// WOLMessage 唤醒消息，仅控制端向被控端下发
type WOLMessage struct {
	MacAddress string `json:"mac_address"`
}
