# FD协议

FD协议是基于MQTT的点对点任务分发协议。节点间直接通信，无需中心服务器调度。

**核心特性:**
- 点对点通信 - 节点直接发送任务给目标节点
- 消息可靠 - QoS 1 + 持久会话保证离线消息不丢失
- 任务追踪 - 通过 ack/complete/failed 跟踪任务生命周期
- 访问控制 - ACL 规则确保节点只能访问自己的主题

**适用场景:** 分布式任务调度、设备远程控制、微服务间通信

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

## 通信模型解析

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

### 不是监听"别人的主题"

这不是监听"别人的主题"：
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

### MessageStatus (状态消息)

状态消息使用 MQTT 保留消息(Retained Message)发布到 `nodes/{username}/status` 主题。
每个节点定期更新自己的状态,其他节点可以订阅该主题获取最新状态。

```go
type MessageStatus struct {
    Time     int64  `json:"time"`               // 状态更新时间戳(秒)
    Online   *bool  `json:"online,omitempty"`   // 在线状态(可选)
    Battery  *int   `json:"battery,omitempty"`  // 电量百分比 0-100(可选)
    Rssi     *int   `json:"rssi,omitempty"`     // 信号强度 dBm(可选)
    Ip       string `json:"ip,omitempty"`       // IP地址(可选)
    Version  string `json:"version,omitempty"`  // 软件版本(可选)
    Uptime   int64  `json:"uptime,omitempty"`   // 运行时长(秒)(可选)
    Data     []byte `json:"data,omitempty"`     // 其他业务自定义数据(可选)
}
```

**状态消息特点:**
- 使用保留消息(Retained=true),新订阅者立即收到最新状态
- 节点定期发布更新自己的状态
- 其他节点订阅 `nodes/{target_username}/status` 获取目标节点状态
- 发送空载荷可删除保留消息
- 所有字段(除Time外)都是可选的,由业务根据实际需求选择使用

**字段说明:**
- `Sender`: 发送者的 MQTT username，回复消息时需要发布到 `nodes/{sender}/ack` 等主题，但此字段可被伪造，需额外验证
- `Receiver`: 接收者的 MQTT username，用于确定消息发布到哪个主题 `nodes/{receiver}/pending`，应用层无需关注
- `MsgId`: 关联请求和响应,建议使用UUID
- `Time`: 消息创建时间,可用于检测时间偏差
- `Exp`: 过期时间,接收方应丢弃过期消息,不执行该任务,也不需要ask或failed
- `Payload`: 实际业务数据,可根据 `Action` 字段进行区分解码

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

## 安全建议
- 本协议只规定了数据流向，未保证第三者伪造发信人，Payload中需要自行鉴别发信人身份。
- 使用ACL确保客户端只能订阅和发布到自己username相关的主题

## EMQX ACL 规则参考

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

## 遗嘱消息 (Last Will and Testament)

遗嘱消息用于在节点异常断线时自动通知其他节点。当节点连接到 MQTT Broker 时设置遗嘱消息,如果节点异常断开(网络故障、程序崩溃等),Broker 会自动发布遗嘱消息。

### 遗嘱消息配置

**主题:** `nodes/{username}/status`
**QoS:** 1
**Retained:** true (保留消息)
**Payload:**
```json
{
  "time": 1234567890,
  "online": false
}
```

### 工作机制

1. **连接时设置遗嘱** - 节点连接 MQTT Broker 时配置遗嘱消息
2. **正常断开不触发** - 节点正常断开连接(发送 DISCONNECT)时,遗嘱消息不会发布
3. **异常断开触发** - 节点异常断开(网络故障、超时、崩溃)时,Broker 自动发布遗嘱消息
4. **状态同步** - 遗嘱消息发布到 status 主题,其他节点订阅该主题可立即感知节点离线

### 最佳实践

1. **正常上线时发布在线状态**
   ```json
   {
     "time": 1234567890,
     "online": true,
     "ip": "192.168.1.100",
     "version": "1.0.0"
   }
   ```

2. **定期更新状态** - 节点运行期间定期发布状态更新(如每分钟一次)

3. **正常下线时发布离线状态** - 节点正常退出前主动发布离线状态,不依赖遗嘱消息
   ```json
   {
     "time": 1234567890,
     "online": false
   }
   ```

4. **遗嘱消息作为兜底** - 遗嘱消息确保异常情况下其他节点也能感知离线

## 持久会话

使用持久会话特性，低功耗设备可以定时休眠，例如5分钟唤醒一次联网，而不会错过消息

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

使用保留消息特性，即使设备离线，应用端仍可以获取到设备最后的状态

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

