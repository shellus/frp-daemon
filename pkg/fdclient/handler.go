package fdclient

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/shellus/frp-daemon/pkg/types"
)

func (c *Client) HandlePing(action string, payload []byte) (value []byte, err error) {
	var ping types.PingMessage
	if err = json.Unmarshal(payload, &ping); err != nil {
		c.logger.Error().Msgf("处理ping指令解析失败，Error=%v", err)
		return
	}
	now := time.Now().UnixMilli()
	c.logger.Info().Msgf("处理心跳消息，单向延迟=%dms", now-ping.Time)
	pingBytes, err := json.Marshal(types.PingMessage{
		Time: now,
	})
	if err != nil {
		c.logger.Error().Msgf("序列化心跳消息失败: %v", err)
		return
	}
	return pingBytes, nil
}

// HandleUpdate 处理下发frp实例
func (c *Client) HandleUpdate(action string, payload []byte) (value []byte, err error) {
	var instance types.InstanceConfigRemote
	if err = json.Unmarshal(payload, &instance); err != nil {
		return nil, fmt.Errorf("处理update指令解析失败，Error=%v", err)
	}
	c.logger.Info().Msgf("处理update指令，instanceName=%s, version=%s", instance.Name, instance.Version)

	// 验证密码
	if instance.ClientPassword != c.configFile.ClientConfig.Client.Password {
		c.logger.Info().Msgf("验证密码失败拒绝更新，instanceName=%s, version=%s", instance.Name, instance.Version)
		return nil, fmt.Errorf("验证密码失败拒绝更新，instanceName=%s, version=%s", instance.Name, instance.Version)
	}

	// 配置写入本地文件得到文件名
	filePath := fmt.Sprintf("%s/%s.yaml", c.instancesDir, instance.Name)
	// 写入文件
	err = os.WriteFile(filePath, []byte(instance.ConfigContent), 0644)
	if err != nil {
		c.logger.Error().Msgf("写入frpc.ini配置失败，Error=%v", err)
		return nil, fmt.Errorf("写入frpc.ini配置失败，Error=%v", err)
	}
	c.logger.Info().Msgf("写入frpc.ini配置成功，filePath=%s", filePath)

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
		c.logger.Error().Msgf("启动实例失败，instanceName=%s, Error=%v", localInstance.Name, err)
		return nil, fmt.Errorf("启动实例失败，instanceName=%s, Error=%v", localInstance.Name, err)
	}

	// 更新持久化实例配置
	err = c.configFile.UpdateInstance(localInstance)
	if err != nil {
		c.logger.Error().Msgf("更新实例配置失败，Error=%v", err)
		return nil, fmt.Errorf("更新实例配置失败，Error=%v", err)
	}

	c.logger.Info().Msgf("处理update指令完成，instanceName=%s", localInstance.Name)

	respByte, err := json.Marshal("搞完了")
	if err != nil {
		return nil, fmt.Errorf("序列化响应失败，Error=%v", err)
	}
	return respByte, nil
}
