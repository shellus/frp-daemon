package types

import (
	"crypto/rand"
	"encoding/hex"
)

// generateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}



const (
	// MessageActionPing 对应的Payload是PingMessage
	MessageActionPing string = "ping"
	// MessageActionUpdate 对应的Payload是InstanceConfigRemote
	MessageActionUpdate string = "update"
	// MessageActionDelete 对应的Payload是DeleteInstanceMessage
	MessageActionDelete string = "delete"
	// MessageActionGetStatus 对应的Payload是GetStatusMessage
	MessageActionGetStatus string = "get_status"
	// MessageActionWOL 对应的Payload是WOLMessage
	MessageActionWOL string = "wol"
)