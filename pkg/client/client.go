package client

import (
	"github.com/shellus/frp-daemon/pkg/mqtt"
	"github.com/shellus/frp-daemon/pkg/types"
)

type Client struct {
	auth *types.ClientAuth
	mqttConfig *types.MQTTClientOpts
	mqtt *mqtt.MQTT
}

func NewClient(auth *types.ClientAuth, mqttConfig *types.MQTTClientOpts) (*Client, error) {
	client := &Client{
		auth: auth,
		mqttConfig: mqttConfig,
	}

	mqtt := mqtt.NewMQTT(mqttConfig)
	if err := mqtt.Connect(); err != nil {
		return nil, err
	}

	client.mqtt = mqtt
	// 开始监听
	go client.listen()

	return client, nil
}

func (c *Client) listen() {
	for {
		select {}
	}
}
