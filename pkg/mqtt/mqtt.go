package mqtt

import (
	"errors"
	"fmt"
	"time"

	"encoding/json"
	"log"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/shellus/frp-daemon/pkg/types"
)

type MessageHandler func(message types.Message)

type MQTT struct {
	config types.MQTTClientOpts
	client pahomqtt.Client
}

func NewMQTT(config types.MQTTClientOpts) (*MQTT, error) {
	if config.Broker == "" {
		return nil, errors.New("mqtt config broker is empty")
	}
	return &MQTT{config: config}, nil
}

func (m *MQTT) Connect() error {
	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(m.config.Broker)
	opts.SetUsername(m.config.Username)
	opts.SetPassword(m.config.Password)
	opts.SetClientID(m.config.ClientID)
	opts.SetCleanSession(m.config.CleanSession)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(1 * time.Second)

	// 添加连接超时
	opts.SetConnectTimeout(10 * time.Second)
	// 设置最大重连次数，避免无限等待
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(60 * time.Second)

	// 添加连接回调，处理连接失败的情况
	opts.SetConnectionLostHandler(func(client pahomqtt.Client, err error) {
		log.Printf("MQTT连接断开: %v", err)
	})

	opts.SetOnConnectHandler(func(client pahomqtt.Client) {
		log.Printf("MQTT连接成功")
	})

	client := pahomqtt.NewClient(opts)
	connToken := client.Connect()

	// 等待连接完成，设置超时
	if !connToken.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("MQTT连接超时，请检查服务器地址或网络连接：%s", m.config.Broker)
	}

	if connToken.Error() != nil {
		return fmt.Errorf("MQTT连接失败，请检查用户名密码是否正确：%s", connToken.Error())
	}

	m.client = client
	return nil
}

func (m *MQTT) Disconnect() error {
	m.client.Disconnect(250)
	return nil
}

func (m *MQTT) Publish(topic string, message types.Message, qos byte, retain bool) error {
	// 序列化消息为JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message failed: %v", err)
	}

	token := m.client.Publish(topic, qos, retain, jsonData)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTT) Subscribe(topic string, qos byte, callback MessageHandler) error {
	token := m.client.Subscribe(topic, qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		var message types.Message
		mqttMessage := msg.Payload()
		if err := json.Unmarshal(mqttMessage, &message); err != nil {
			log.Printf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		callback(message)
	})
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTT) Unsubscribe(topic string) error {
	token := m.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
