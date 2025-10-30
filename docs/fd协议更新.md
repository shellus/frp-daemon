# FD协议更新指南

## 更新日期
2025-10-30

## 概述
本次更新简化了FD协议的消息结构体字段命名,使其更加简洁易用。此更新影响所有使用FD协议的项目。

## 变更内容

### 1. 消息结构体字段重命名

#### MessagePending (任务消息)

**旧字段名 → 新字段名:**
```
SenderClientId   → Sender
ReceiverClientId → Receiver
MessageId        → MsgId
Timestamp        → Time
Expiration       → Exp
Action           → Action   (保持不变)
Payload          → Payload  (保持不变)
```

**旧结构体定义:**
```go
type MessagePending struct {
    SenderClientId   string `json:"sender_client_id"`   // 发送者客户端ID
    ReceiverClientId string `json:"receiver_client_id"` // 接收者客户端ID
    MessageId        string `json:"message_id"`         // 消息ID
    Action           string `json:"action"`             // 消息动作
    Timestamp        int64  `json:"timestamp"`          // 消息发送时间戳，单位为秒
    Expiration       int64  `json:"expiration"`         // 消息过期时间戳，单位为秒
    Payload          []byte `json:"payload"`            // 消息负载
}
```

**新结构体定义:**
```go
type MessagePending struct {
    Sender   string `json:"sender"`   // 发送者
    Receiver string `json:"receiver"` // 接收者
    MsgId    string `json:"msg_id"`   // 消息ID，UUID格式
    Action   string `json:"action"`   // 消息动作类型，内容自定义，例如"开灯"
    Time     int64  `json:"time"`     // 消息发送时间戳(秒)
    Exp      int64  `json:"exp"`      // 消息过期时间戳(秒)
    Payload  []byte `json:"payload"`  // 消息负载(业务数据)
}
```

#### MessageAck (确认消息)

**旧字段名 → 新字段名:**
```
MessageId → MsgId
```

**旧结构体定义:**
```go
type MessageAsk struct {
    MessageId string `json:"message_id"`
}
```

**新结构体定义:**
```go
type MessageAck struct {
    MsgId string `json:"msg_id"` // 对应的任务消息ID
}
```

#### MessageComplete (完成消息)

**旧字段名 → 新字段名:**
```
MessageId → MsgId
Value     → Value (保持不变)
```

**旧结构体定义:**
```go
type MessageComplete struct {
    MessageId string `json:"message_id"`
    Value     []byte `json:"value"`
}
```

**新结构体定义:**
```go
type MessageComplete struct {
    MsgId string `json:"msg_id"` // 对应的任务消息ID
    Value []byte `json:"value"`  // 返回值
}
```

#### MessageFailed (失败消息)

**旧字段名 → 新字段名:**
```
MessageId → MsgId
Error     → Error (保持不变)
```

**旧结构体定义:**
```go
type MessageFailed struct {
    MessageId string `json:"message_id"`
    Error     []byte `json:"error"`
}
```

**新结构体定义:**
```go
type MessageFailed struct {
    MsgId string `json:"msg_id"` // 对应的任务消息ID
    Error []byte `json:"error"`  // 错误信息
}
```

### 2. 字段说明更新

**Sender (原 SenderClientId):**
- 发送者的 MQTT username
- 回复消息时需要发布到 `nodes/{sender}/ack` 等主题
- 此字段可被伪造,需额外验证

**Receiver (原 ReceiverClientId):**
- 接收者的 MQTT username
- 用于确定消息发布到哪个主题 `nodes/{receiver}/pending`
- 应用层无需关注

**MsgId (原 MessageId):**
- 关联请求和响应
- 建议使用UUID

**Time (原 Timestamp):**
- 消息创建时间
- 可用于检测时间偏差

**Exp (原 Expiration):**
- 过期时间
- 接收方应丢弃过期消息

**重要说明:**
- `Sender` 和 `Receiver` 字段的值就是 MQTT 连接时使用的 username
- 主题结构为 `nodes/{username}/{action}`，其中 `{username}` 对应 `Sender` 或 `Receiver` 字段

## 迁移指南

### 步骤1: 更新结构体定义

在 `pkg/mqtt/task/tasks.go` 或对应的消息定义文件中,更新所有消息结构体:

```go
// 更新 MessagePending
type MessagePending struct {
    Sender   string `json:"sender"`
    Receiver string `json:"receiver"`
    MsgId    string `json:"msg_id"`
    Action   string `json:"action"`
    Time     int64  `json:"time"`
    Exp      int64  `json:"exp"`
    Payload  []byte `json:"payload"`
}

// 更新 MessageAck
type MessageAck struct {
    MsgId string `json:"msg_id"`
}

// 更新 MessageComplete
type MessageComplete struct {
    MsgId string `json:"msg_id"`
    Value []byte `json:"value"`
}

// 更新 MessageFailed
type MessageFailed struct {
    MsgId string `json:"msg_id"`
    Error []byte `json:"error"`
}
```

### 步骤2: 更新代码中的字段访问

使用全局搜索替换功能,更新所有字段访问:

**搜索替换列表:**
```
.SenderClientId   → .Sender
.ReceiverClientId → .Receiver
.MessageId        → .MsgId
.Timestamp        → .Time
.Expiration       → .Exp
```

**注意:** 使用正则表达式或IDE的"全词匹配"功能,避免误替换。

### 步骤3: 更新消息构造代码

**旧代码示例:**
```go
msg := task.MessagePending{
    SenderClientId:   "node_A",  // 发送者的 MQTT username
    ReceiverClientId: "node_B",  // 接收者的 MQTT username
    MessageId:        uuid.New().String(),
    Action:           "test",
    Timestamp:        time.Now().Unix(),
    Expiration:       time.Now().Add(1*time.Hour).Unix(),
    Payload:          []byte("{}"),
}
// 消息会发布到主题: nodes/node_B/pending
```

**新代码示例:**
```go
msg := task.MessagePending{
    Sender:   "node_A",  // 发送者的 MQTT username
    Receiver: "node_B",  // 接收者的 MQTT username
    MsgId:    uuid.New().String(),
    Action:   "test",
    Time:     time.Now().Unix(),
    Exp:      time.Now().Add(1*time.Hour).Unix(),
    Payload:  []byte("{}"),
}
// 消息会发布到主题: nodes/node_B/pending
```

### 步骤4: 更新JSON序列化/反序列化

由于JSON tag已更新,确保所有JSON序列化和反序列化的代码都能正常工作。

**测试示例:**
```go
// 序列化测试
msg := task.MessagePending{
    Sender:   "A",
    Receiver: "B",
    MsgId:    "msg001",
    Action:   "test",
    Time:     1234567890,
    Exp:      9999999999,
    Payload:  []byte("{}"),
}
jsonData, _ := json.Marshal(msg)
fmt.Println(string(jsonData))
// 期望输出: {"sender":"A","receiver":"B","msg_id":"msg001","action":"test","time":1234567890,"exp":9999999999,"payload":"e30="}

// 反序列化测试
var msg2 task.MessagePending
json.Unmarshal(jsonData, &msg2)
fmt.Printf("%+v\n", msg2)
```

### 步骤5: 更新测试用例

更新所有单元测试和集成测试中的字段名:

```go
// 旧测试代码
assert.Equal(t, "node_A", msg.SenderClientId)   // MQTT username
assert.Equal(t, "node_B", msg.ReceiverClientId) // MQTT username
assert.Equal(t, "msg001", msg.MessageId)

// 新测试代码
assert.Equal(t, "node_A", msg.Sender)   // MQTT username
assert.Equal(t, "node_B", msg.Receiver) // MQTT username
assert.Equal(t, "msg001", msg.MsgId)
```

## 影响范围

### 需要更新的文件类型

1. **结构体定义文件**
   - `pkg/mqtt/task/tasks.go`
   - 或其他定义消息结构体的文件

2. **消息处理代码**
   - `pkg/mqtt/mqtt.go` - MQTT消息处理
   - `pkg/fdctl/controller.go` - 控制端消息发送
   - `pkg/fdclient/client.go` - 客户端消息处理
   - `pkg/fdclient/handler.go` - 消息处理器

3. **测试文件**
   - 所有包含消息结构体测试的文件

### 搜索关键字

使用以下关键字在项目中搜索需要更新的位置:

```
SenderClientId
ReceiverClientId
MessageId
Timestamp
Expiration
sender_client_id
receiver_client_id
message_id
timestamp
expiration
```

## 兼容性说明

### 向后兼容性
**不兼容!** 此更新修改了JSON字段名,旧版本和新版本之间无法直接通信。

### 升级策略

**方案1: 全量升级 (推荐)**
- 同时升级所有节点到新版本
- 适用于节点数量较少的场景

**方案2: 灰度升级**
- 暂时保留两套字段,同时支持新旧格式
- 在反序列化时尝试两种格式
- 所有节点升级完成后,移除旧字段支持

**灰度升级示例代码:**
```go
type MessagePending struct {
    // 新字段
    Sender   string `json:"sender"`
    Receiver string `json:"receiver"`
    MsgId    string `json:"msg_id"`
    Time     int64  `json:"time"`
    Exp      int64  `json:"exp"`
    
    // 旧字段 (兼容期保留)
    SenderClientIdOld   string `json:"sender_client_id,omitempty"`
    ReceiverClientIdOld string `json:"receiver_client_id,omitempty"`
    MessageIdOld        string `json:"message_id,omitempty"`
    TimestampOld        int64  `json:"timestamp,omitempty"`
    ExpirationOld       int64  `json:"expiration,omitempty"`
    
    Action  string `json:"action"`
    Payload []byte `json:"payload"`
}

// 反序列化后的兼容处理
func (m *MessagePending) Normalize() {
    if m.Sender == "" && m.SenderClientIdOld != "" {
        m.Sender = m.SenderClientIdOld
    }
    if m.Receiver == "" && m.ReceiverClientIdOld != "" {
        m.Receiver = m.ReceiverClientIdOld
    }
    if m.MsgId == "" && m.MessageIdOld != "" {
        m.MsgId = m.MessageIdOld
    }
    if m.Time == 0 && m.TimestampOld != 0 {
        m.Time = m.TimestampOld
    }
    if m.Exp == 0 && m.ExpirationOld != 0 {
        m.Exp = m.ExpirationOld
    }
}
```

## 验证清单

更新完成后,请验证以下内容:

- [ ] 所有结构体定义已更新
- [ ] 所有字段访问代码已更新
- [ ] 所有消息构造代码已更新
- [ ] 所有测试用例已更新并通过
- [ ] JSON序列化/反序列化正常工作
- [ ] 与其他节点的通信正常
- [ ] 日志输出中的字段名已更新

## 参考资料

- FD协议文档: `pkg/mqtt/README.md`
- 消息结构体定义: `pkg/mqtt/task/tasks.go`
- MQTT通信实现: `pkg/mqtt/mqtt.go`

