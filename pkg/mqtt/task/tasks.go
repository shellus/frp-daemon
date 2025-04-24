package task

import (
	"fmt"
)

// MQTT的任务主题默认配置

// 保留消息由发送方在publish时决定，一个主题只保留1个保留消息，并且保留消息不会因为接收而被消费
// 只有发送方再次发送一个相同主题的消息时，保留消息才会被覆盖。
// 删除主题上的保留消息，发送方可以发送一个空载荷（empty payload）的保留消息到该主题，否则其他情况下，该主题上的保留消息不会消失。
// 这意味着保留消息适合用于状态通知、属性上报等场景。而不能用于任务下发，因为会重复覆盖，并且不会因接收消费而被删除。

// CleanSession 的含义是，当客户端断开连接后，代理是否清除会话
// 如果不清除，那么代理仍然会将该客户端被订阅到的消息缓存，当客户端重新连接时，代理会继续发送这些消息给这个订阅的客户端。
// 所以持久性消息适合用于任务下发，消息传递等场景，可以保证消息不丢失。

// 发送消息到目标节点的收件箱：
// tasks/{target_username}/pending
// 订阅消息结果：
// tasks/{self_username}/ack
// tasks/{self_username}/complete
// tasks/{self_username}/failed

// 订阅发给自己的指令
// tasks/{self_username}/pending - 接收分配给自己的新任务
// 发送消息结果到发送方：
// tasks/{from_username}/ack - 确认已收到任务
// tasks/{from_username}/complete - 报告任务成功完成
// tasks/{from_username}/failed - 报告任务执行失败

// 使用保留消息发布，上报自己的信息、状态
// tasks/{self_username}/status - 上报自己的状态

type MessagePending struct {
	SenderClientId   string `json:"sender_client_id"`   // 发送者客户端ID
	ReceiverClientId string `json:"receiver_client_id"` // 接收者客户端ID
	MessageId        string `json:"message_id"`         // 消息ID，一般为UUID，resp和req的message_id相同
	Action           string `json:"action"`             // 消息动作
	Timestamp        int64  `json:"timestamp"`          // 消息发送时间戳，单位为秒
	Expiration       int64  `json:"expiration"`         // 消息过期时间戳，单位为秒
	Payload          []byte `json:"payload"`            // 消息负载
}
type MessageAsk struct {
	MessageId string `json:"message_id"`
}
type MessageComplete struct {
	MessageId string `json:"message_id"`
	Value     []byte `json:"value"`
}
type MessageFailed struct {
	MessageId string `json:"message_id"`
	Error     []byte `json:"error"`
}

func TopicPending(prefix string, username string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, username, "pending")
}

func TopicAsk(prefix string, username string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, username, "ack")
}

func TopicComplete(prefix string, username string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, username, "complete")
}

func TopicFailed(prefix string, username string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, username, "failed")
}

func TopicStatus(prefix string, username string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, username, "status")
}
