package emqx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shellus/frp-daemon/pkg/types"
)

// 文档地址：https://docs.emqx.com/en/emqx/v5.8/admin/api-docs.html

// API EMQX API客户端
type API struct {
	config types.EMQXAPIConfig
}

// NewAPI 创建EMQX API客户端
func NewAPI(config types.EMQXAPIConfig) *API {
	return &API{
		config: config,
	}
}

// CreateUser 创建MQTT用户，返回MQTT配置
func (a *API) CreateUser(auth *types.ClientAuth) (*types.MQTTClientOpts, error) {
	url := fmt.Sprintf("%s/api/v5/authentication/%s/users", a.config.ApiEndpoint, "password_based:built_in_database")

	mqttClientId := auth.ClientId
	mqttPassword := auth.Password
	// 准备请求体
	body := map[string]interface{}{
		"user_id":      mqttClientId,
		"password":     mqttPassword,
		"is_superuser": false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(a.config.ApiAppKey, a.config.ApiSecretKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	var respJson map[string]interface{}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}
	err = json.Unmarshal(bodyBytes, &respJson)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("创建用户失败：状态码: %d", resp.StatusCode)
	}

	// 返回MQTT配置
	return &types.MQTTClientOpts{
		Broker:      a.config.MQTTBroker,
		ClientID:    auth.Name,
		Username:    mqttClientId,
		Password:    mqttPassword,
		TopicPrefix: types.TopicPrefix,
	}, nil
}

// DeleteUser 删除MQTT用户
func (a *API) DeleteUser(auth *types.ClientAuth) error {
	url := fmt.Sprintf("%s/api/v5/authentication/%s/users/%s", a.config.ApiEndpoint, "password_based:built_in_database", auth.ClientId)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(a.config.ApiAppKey, a.config.ApiSecretKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("删除用户失败：状态码: %d", resp.StatusCode)
	}

	return nil
}
