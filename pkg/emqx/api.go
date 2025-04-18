package emqx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/shellus/frp-daemon/pkg/types"
)

// API EMQX API客户端
type API struct {
	config *types.EMQXAPIConfig
}

// NewAPI 创建EMQX API客户端
func NewAPI(config *types.EMQXAPIConfig) *API {
	return &API{
		config: config,
	}
}

// CreateUser 创建MQTT用户，返回MQTT配置
func (a *API) CreateUser(auth *types.ClientAuth) (*types.MQTTClientOpts, error) {
	url := fmt.Sprintf("%s/api/v5/authentication/%s/users", a.config.ApiEndpoint, "password_based:built_in_database")

	mqttClientId := types.GenerateRandomString(16)
	mqttPassword := types.GenerateRandomString(32)
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
		log.Printf("解析响应失败：curl=%s, body=%s，resp=%+v", url, string(jsonBody), respJson)
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusCreated {
		log.Printf("创建用户失败：curl=%s, body=%s，resp=%+v", url, string(jsonBody), respJson)
		return nil, fmt.Errorf("创建用户失败：状态码: %d", resp.StatusCode)
	}

	// 返回MQTT配置
	return &types.MQTTClientOpts{
		Broker:       a.config.MQTTBroker,
		ClientID:     auth.ID,
		Username:     mqttClientId,
		Password:     mqttPassword,
		TopicPrefix:  types.TopicPrefix,
		QoS:          types.QoS,
		Retain:       types.Retain,
		CleanSession: types.CleanSession,
	}, nil
}
