package mqtt

import (
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/shellus/frp-daemon/pkg/types"
)

type MQTT struct {
	config *types.MQTTClientOpts
	client pahomqtt.Client
}

func NewMQTT(config *types.MQTTClientOpts) *MQTT {
	return &MQTT{config: config}
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

	client := pahomqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	m.client = client

	return nil
}

func (m *MQTT) Disconnect() error {
	m.client.Disconnect(250)
	return nil
}

func (m *MQTT) Publish(topic string, payload []byte, qos byte, retain bool) error {
	token := m.client.Publish(topic, qos, retain, payload)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTT) Subscribe(topic string, qos byte, callback pahomqtt.MessageHandler) error {
	token := m.client.Subscribe(topic, qos, callback)
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
