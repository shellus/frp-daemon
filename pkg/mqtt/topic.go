package mqtt

import "fmt"

// MessageTopic 等待消息的主题
func MessageTopic(prefix string, clientId string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, clientId, "messages")
}

// ReplyTopic 回复消息的主题
func ReplyTopic(prefix string, clientId string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, clientId, "reply")
}

func Topic(prefix string, clientId string, action string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, clientId, action)
}
