package fdclient

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shellus/frp-daemon/pkg/types"
)

// existsInstance 检查实例是否存在
func existsInstance(instances []types.InstanceConfigLocal, name string) bool {
	for _, instance := range instances {
		if instance.Name == name {
			return true
		}
	}
	return false
}

func (c *Client) HandlePing(action string, payload []byte) (value []byte, err error) {
	var ping types.PingMessage
	if err = json.Unmarshal(payload, &ping); err != nil {
		log.Printf("处理ping指令解析失败，Error=%v", err)
		return
	}
	now := time.Now().UnixMilli()
	log.Printf("处理心跳消息，单向延迟=%dms", now-ping.Time)
	pingBytes, err := json.Marshal(types.PingMessage{
		Time: now,
	})
	if err != nil {
		log.Printf("序列化心跳消息失败: %v", err)
		return
	}
	return pingBytes, nil
}

// HandleUpdate 处理下发frp实例
func (c *Client) HandleUpdate(action string, payload []byte) (value []byte, err error) {
	var instance types.InstanceConfigRemote
	if err = json.Unmarshal(payload, &instance); err != nil {
		log.Printf("处理update指令解析失败，Error=%v", err)
		return
	}
	log.Printf("处理update指令，instanceName=%s, version=%s", instance.Name, instance.Version)

	// 配置写入本地文件得到文件名
	filePath := fmt.Sprintf("%s/%s.yaml", c.instancesDir, instance.Name)
	// 写入文件
	err = os.WriteFile(filePath, []byte(instance.ConfigContent), 0644)
	if err != nil {
		log.Printf("写入frpc.ini配置失败，Error=%v", err)
		return
	}
	log.Printf("写入frpc.ini配置成功，filePath=%s", filePath)

	// 生成本地实例配置
	localInstance := types.InstanceConfigLocal{
		Name:       instance.Name,
		Version:    instance.Version,
		ConfigPath: filePath,
	}

	// 如果存在先停止，那就不管错误了。
	c.StopFrpInstance(localInstance.Name)

	// 启动实例
	if err = c.StartFrpInstance(localInstance); err != nil {
		log.Printf("启动实例失败，instanceName=%s, Error=%v", localInstance.Name, err)
		return
	}

	// 更新持久化实例配置
	err = c.configFile.UpdateInstance(localInstance)
	if err != nil {
		log.Printf("更新实例配置失败，Error=%v", err)
		return
	}

	log.Printf("处理update指令完成，instanceName=%s", localInstance.Name)

	respByte, err := json.Marshal("搞完了")
	if err != nil {
		return
	}
	return respByte, nil
}
