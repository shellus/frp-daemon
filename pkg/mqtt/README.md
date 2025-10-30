# FD协议
fd协议是一个对等协议，没有特权服务器，只有特定功能的对等终端，例如目录服务终端，服务发现终端等

在传统的cs、bs架构中，客户端连接到服务器，然后什么都听从服务器的安排。

在fd协议中，客户端信任一些终端ID，然后从它们那里使用服务，例如信任一个目录服务客户端，然后找它得到另一个客户端的信息，并和另一个客户端直接联系。

当前版本基于MQTT实现，依赖固定的MQTT代理服务端。


## 基于MQTT的同步、异步指令下发架构
1. 使用QoS 1（至少一次）确保消息送达，每个终端使用持久会话连接（Clean Session = false）确保离线期间的消息在客户端重连后能够接收
2. 调用方发布任务到`pending`主题，终端订阅`pending`主题接收到任务回复`ack`，调用方收到`ack`确认送达，调用方可通过`complete`、`failed`主题跟踪每个任务的状态和生命周期

## 主题（topic）设计
- nodes/{username}/pending    # 向特定节点发送任务
- nodes/{username}/ack        # 任务确认
- nodes/{username}/complete   # 任务完成
- nodes/{username}/failed     # 任务失败
- nodes/{username}/status     # 节点状态

说明: `{username}` 是 MQTT 连接时使用的用户名,每个节点使用唯一的 username 连接到 MQTT Broker。

- 每个节点同时：
- 订阅自己username的入站消息：nodes/{username}/pending
- 订阅自己username的任务响应：nodes/{username}/{ack，complete，failed}
(如果自己不准备接收响应，那就不需要订阅自己的响应主题)

`nodes` 可根据需要换成应用名称或产品名称等

## 节点通信模型解析

这个设计是基于对等节点（P2P）通信模型，其中：

1. **主题结构**: `nodes/{username}/{action}`
   - `{username}` 是接收消息的节点的 MQTT username
   - `{action}` 是操作类型（pending/ack/complete/failed/status）

2. **如何发送消息给其他节点**:
   - 当你要发送任务给节点B时，你发布到 `nodes/B/pending` (B是对方的username)
   - 这意味着你是在"写入对方的信箱"

3. **如何接收别人发给你的消息**:
   - 你需要订阅 `nodes/{你自己的username}/pending`（接收任务）
   - 你需要订阅 `nodes/{你自己的username}/ack`（接收确认）
   - 你需要订阅 `nodes/{你自己的username}/complete`（任务完成）
   - 你需要订阅 `nodes/{你自己的username}/failed`（任务失败）
   - 这相当于"监听自己的信箱"

## 身份验证与主题访问控制：
- 确保客户端只能订阅和发布到自己username相关的主题
- 使用MQTT代理的ACL(访问控制列表)功能实现

## 关于持久会话

### 定义与特性
1. 持久会话（Clean Session = false）是由客户端在连接MQTT代理时决定的，不是由发送方、订阅方或主题决定
2. 持久会话与客户端ID绑定，必须使用相同的客户端ID才能恢复会话
3. 持久会话会保留客户端的订阅信息，重连时无需重新订阅
4. 代理通常对持久会话和离线消息的存储有限制（消息数量、大小、保留时间）
5. 对于发送方来说，只需要发布消息到相应的主题，不需要关心接收方是否在线

### 工作机制
1. 当设置Clean Session = false时，订阅方客户端离线后，代理会为其保存消息
2. 客户端重新连接时（使用相同的客户端ID和Clean Session = false），会收到离线期间的消息
3. 只有QoS级别为1或2的消息会被保存，QoS 0消息不会被保存
4. 如果使用新的客户端ID，将无法接收离线消息
5. 这种机制适合移动设备或IoT设备经常离线的场景，确保设备不会错过重要消息

## 关于QoS
### QoS 1不保证顺序
1. QoS 1（至少一次）保证消息至少送达一次，但不保证顺序
2. 代理和客户端可能会在处理消息时发生重排序

### 常见的QoS 1重复场景
1. 客户端收到消息并处理，但发回的PUBACK确认包在网络中丢失
2. 当网络连接不稳定，断开重连后，代理可能重传未确认的消息
3. MQTT代理在恢复过程中可能无法准确跟踪哪些消息已被确认
4. 如果客户端异常退出，没有正常发送确认，重启后可能再次收到消息
5. 当网络延迟高时，代理可能认为确认超时，并重发消息，而后收到延迟的确认，但重发的消息已发出

在稳定的网络环境中（如数据中心内部网络），重复率通常很低，可能低于1%。而在网络不稳定环境中（如移动网络、卫星连接等），重复率可能达到5-10%

### 应用程序需要设计为能处理重复消息

1. **消息去重**：在消息中包含唯一标识符，记录已处理的消息ID，忽略重复消息
2. **幂等操作**：设计处理逻辑使其多次执行同一消息不会产生副作用，例如，"设置温度为25度"而不是"增加温度1度"
3. **消息时间戳**：为消息添加时间戳，丢弃超过特定时间窗口的消息
4. **序列号**：使用递增的序列号标记消息，忽略序列号小于或等于已处理消息的新消息

如果应用对消息重复非常敏感，可以考虑使用QoS 2（恰好一次），但这会增加通信开销和延迟。

总的来说，QoS 1消息重复是正常且预期的行为，在设计基于MQTT的系统时应该将其考虑在内。

## 关于保留消息

### 定义与特性
1. 保留消息（Retained Message）是由发送方决定的，而不是订阅方
2. 每个主题只能有一个保留消息，新的保留消息会替换旧的
3. 保留消息由代理存储，直到被新的保留消息替换或被明确删除
4. 订阅收到保留消息后，客户端无法自动"消费"掉该消息
5. 这种机制特别适合发布设备温度、开关状态、配置参数等，订阅者需要知道最新状态，而不关心历史状态变化过程

### 工作机制
1. 在发布消息时设置"retain"标志为true，告诉MQTT代理保存该消息的最新版本
2. 当新客户端订阅包含保留消息的主题时，它会立即收到该主题的最新保留消息
3. 要删除一个保留消息，发送方可以发送一个空载荷（empty payload）的保留消息到该主题


## mosquitto 示例

假设 MQTT Broker 地址为 `broker.emqx.io:1883`，节点A向节点B发送一个任务并获取回复：

### 终端1 - 节点B订阅自己的 pending 主题（接收任务）
```bash
mosquitto_sub -h broker.emqx.io -p 1883 -t "nodes/B/pending" -q 1 -u B -P password_b
```

### 终端2 - 节点A订阅自己的 complete 主题（接收完成响应）
```bash
mosquitto_sub -h broker.emqx.io -p 1883 -t "nodes/A/complete" -q 1 -u A -P password_a
```

### 终端3 - 节点A发送任务给节点B
```bash
mosquitto_pub -h broker.emqx.io -p 1883 -t "nodes/B/pending" -q 1 -u A -P password_a \
  -m '{"sender":"A","receiver":"B","msg_id":"msg001","action":"test","time":1234567890,"exp":9999999999,"payload":{}}'
```

此时终端1（节点B）会收到任务消息。

### 终端4 - 节点B回复完成消息给节点A
```bash
mosquitto_pub -h broker.emqx.io -p 1883 -t "nodes/A/complete" -q 1 -u B -P password_b \
  -m '{"msg_id":"msg001","value":"task completed successfully"}'
```

### 不是监听"别人的主题"

这不是监听"别人的主题"，而是：
1. 每个节点都有自己的"命名空间"（`nodes/{username}/+`）
2. 你订阅自己的命名空间来接收消息
3. 你发布到对方的命名空间来发送消息

这种设计的优点是清晰的数据流向和简单的访问控制，因为每个节点只需权限访问自己的接收主题和其他节点的发送主题。

这实际上类似于每个人都有一个邮箱，你往别人的邮箱里放信，同时查看自己的邮箱收信 - 而不是直接进入别人家看他们是否有你的信。

## 消息结构体

### MessagePending (任务消息)
```go
type MessagePending struct {
    Sender   string `json:"sender"`   // 发送者
    Receiver string `json:"receiver"` // 接收者
    MsgId    string `json:"msg_id"`   // 消息ID，UUID格式
    Action   string `json:"action"`   // 消息动作类型，内容自定义，例如“开灯”
    Time     int64  `json:"time"`     // 消息发送时间戳(秒)
    Exp      int64  `json:"exp"`      // 消息过期时间戳(秒)
    Payload  []byte `json:"payload"`  // 消息负载(业务数据)
}
```

### MessageAck (确认消息)
```go
type MessageAck struct {
    MsgId string `json:"msg_id"` // 对应的任务消息ID
}
```

### MessageComplete (完成消息)
```go
type MessageComplete struct {
    MsgId string `json:"msg_id"` // 对应的任务消息ID
    Value []byte `json:"value"`  // 返回值
}
```

### MessageFailed (失败消息)
```go
type MessageFailed struct {
    MsgId string `json:"msg_id"` // 对应的任务消息ID
    Error []byte `json:"error"`  // 错误信息
}
```

**字段说明:**
- `Sender`: 发送者的 MQTT username，回复消息时需要发布到 `nodes/{sender}/ack` 等主题，但此字段可被伪造，需额外验证
- `Receiver`: 接收者的 MQTT username，用于确定消息发布到哪个主题 `nodes/{receiver}/pending`，应用层无需关注
- `MsgId`: 关联请求和响应,建议使用UUID
- `Time`: 消息创建时间,可用于检测时间偏差
- `Exp`: 过期时间,接收方应丢弃过期消息,不执行该任务,也不需要ask或failed
- `Payload`: 实际业务数据,可根据 `Action` 字段进行区分解码

## 安全建议
本协议只规定了数据的流向，并不负责“信件”是否是第三者伪造发信人，所以payload中需要自行鉴别发信人身份。

## EMQX ACL 规则建议

```
%% 通信模型
%% 自己只可以订阅自己的 pending (接收任务)
%% 自己只可以订阅自己的 ack/complete/failed (接收任务响应)
%% 任何人可以发布到任何节点的 pending (发任务)
%% 任何人可以发布到任何节点的 ack/complete/failed (回复任务)
%% 自己只可以发布自己的状态 status
%% 任何人可以订阅任何人的状态 status

%% 允许节点订阅自己的 pending 主题
{allow, {username, {re, "^(.+)$"}}, subscribe, ["nodes/$1/pending"]}.

%% 允许节点订阅自己的响应主题
{allow, {username, {re, "^(.+)$"}}, subscribe, ["nodes/$1/ack"]}.
{allow, {username, {re, "^(.+)$"}}, subscribe, ["nodes/$1/complete"]}.
{allow, {username, {re, "^(.+)$"}}, subscribe, ["nodes/$1/failed"]}.

%% 允许节点发布到任何节点的 pending 主题
{allow, all, publish, ["nodes/+/pending"]}.

%% 允许节点发布到任何节点的响应主题
{allow, all, publish, ["nodes/+/ack"]}.
{allow, all, publish, ["nodes/+/complete"]}.
{allow, all, publish, ["nodes/+/failed"]}.

%% 允许节点发布自己的状态
{allow, {username, {re, "^(.+)$"}}, publish, ["nodes/$1/status"]}.

%% 允许任何人订阅任何人的状态
{allow, all, subscribe, ["nodes/+/status"]}.

%% 拒绝其他所有操作
{deny, all}.
```