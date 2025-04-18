package client

import (
	"encoding/json"
	"log"

	mqttC "github.com/shellus/frp-daemon/pkg/mqtt"
	"github.com/shellus/frp-daemon/pkg/types"
)

type Handler interface {
	HandleUpdate(instance types.InstanceConfigRemote)
}

type Client struct {
	auth       types.ClientAuth
	mqttConfig types.MQTTClientOpts
	mqtt       *mqttC.MQTT
	handler    Handler
}

func NewClient(auth types.ClientAuth, mqttConfig types.MQTTClientOpts, handler Handler) (*Client, error) {
	client := &Client{
		auth:       auth,
		mqttConfig: mqttConfig,
		handler:    handler,
	}

	mqtt := mqttC.NewMQTT(mqttConfig)
	if err := mqtt.Connect(); err != nil {
		return nil, err
	}

	client.mqtt = mqtt
	client.mqtt.Subscribe(mqttC.Topic(mqttConfig.TopicPrefix, auth.ClientId, "update"), byte(mqttConfig.QoS), func(jsonMessage []byte) {
		var instance types.InstanceConfigRemote
		if err := json.Unmarshal(jsonMessage, &instance); err != nil {
			log.Printf("解析实例配置失败: %v", err)
			return
		}
		client.handler.HandleUpdate(instance)
	})

	return client, nil
}

func (c *Client) Start() {
	log.Println("客户端启动， 这里似乎应该启动并等待MQTT？")
}
