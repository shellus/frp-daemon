package mqtt

import "fmt"

func Topic(prefix string, clientId string, action string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, clientId, action)
}
