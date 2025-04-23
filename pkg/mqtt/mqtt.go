package mqtt

import (
	"errors"
	"fmt"
	"time"

	"encoding/json"
	"log"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/shellus/frp-daemon/pkg/mqtt/task"
	"github.com/shellus/frp-daemon/pkg/types"
)

type MessageHandler func(action string, payload []byte) (value []byte, err error)

type MQTT struct {
	config       types.MQTTClientOpts
	topicPrefix  string
	qos          byte
	retain       bool
	cleanSession bool
	paho         pahomqtt.Client
	// subscribeActionArr订阅行为调用数组
	subscribeActionArr map[string]MessageHandler
	// waiters 存储等待器
	waiters map[string]*task.Waiter
}

func NewMQTT(config types.MQTTClientOpts) (*MQTT, error) {
	if config.Broker == "" {
		return nil, errors.New("mqtt config broker is empty")
	}
	return &MQTT{
		config:             config,
		topicPrefix:        config.TopicPrefix,
		qos:                1,
		retain:             false,
		cleanSession:       false,
		subscribeActionArr: make(map[string]MessageHandler),
		waiters:            make(map[string]*task.Waiter),
	}, nil
}

func (m *MQTT) Connect() error {
	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(m.config.Broker)
	opts.SetUsername(m.config.Username)
	opts.SetPassword(m.config.Password)
	opts.SetClientID(m.config.ClientID)
	opts.SetCleanSession(m.cleanSession)
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

	m.paho = client
	m.subscribeActionArr = make(map[string]MessageHandler)
	m.registerTaskTopics()
	return nil
}

func (m *MQTT) Disconnect() error {
	m.paho.Disconnect(250)
	return nil
}
func (m *MQTT) SubscribeAction(actionName string, callback MessageHandler) {
	m.subscribeActionArr[actionName] = callback
}

func (m *MQTT) registerTaskTopics() {
	m.subscribe(task.TopicPending(m.topicPrefix, m.config.Username), m.qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		var message task.MessagePending
		mqttMessage := msg.Payload()
		if err := json.Unmarshal(mqttMessage, &message); err != nil {
			log.Printf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		// 这里我都不if判断了，如果别人往这个pending主题胡乱投递不符合task.MessagePending的数据结构，那么后续处理时候自然会报错的。
		m.onTopicPending(message)
	})
	m.subscribe(task.TopicAsk(m.topicPrefix, m.config.Username), m.qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		var message task.MessageAsk
		mqttMessage := msg.Payload()
		if err := json.Unmarshal(mqttMessage, &message); err != nil {
			log.Printf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		m.onTopicAsk(message)
	})
	m.subscribe(task.TopicComplete(m.topicPrefix, m.config.Username), m.qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		var message task.MessageComplete
		mqttMessage := msg.Payload()
		if err := json.Unmarshal(mqttMessage, &message); err != nil {
			log.Printf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		m.onTopicComplete(message)
	})
	m.subscribe(task.TopicFailed(m.topicPrefix, m.config.Username), m.qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		var message task.MessageFailed
		mqttMessage := msg.Payload()
		if err := json.Unmarshal(mqttMessage, &message); err != nil {
			log.Printf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		m.onTopicFailed(message)
	})
}
func (m *MQTT) onTopicPending(msg task.MessagePending) {
	// 先回复一个ask
	askMsg := task.MessageAsk{
		MessageId: msg.MessageId,
	}
	askData, err := json.Marshal(askMsg)
	if err != nil {
		log.Fatalf("行为调用ask序列化失败，Err=%v", err)
		return
	}
	m.publish(task.TopicAsk(m.topicPrefix, msg.SenderClientId), askData, m.qos, m.retain)

	// 从m.subscribeArr找出回调函数并调用，然后将返回值发布到compete或failed
	var callback MessageHandler
	for actionName, v := range m.subscribeActionArr {
		if actionName == string(msg.Action) {
			callback = v
			break
		}
	}
	if callback == nil {
		log.Fatalf("行为调用没有找到回调函数，actionName=%s", msg.Action)
		return
	}
	value, err := callback(string(msg.Action), msg.Payload)
	if err != nil {
		// 回复到失败主题
		failedMsg := task.MessageFailed{
			MessageId: msg.MessageId,
			Error:     json.RawMessage(err.Error()),
		}
		failedData, err := json.Marshal(failedMsg)
		if err != nil {
			log.Fatalf("行为调用ask序列化失败，Err=%v", err)
			return
		}
		m.publish(task.TopicFailed(m.topicPrefix, msg.SenderClientId), failedData, m.qos, m.retain)
	}
	// 回复到完成主题
	complepeMsg := task.MessageComplete{
		MessageId: msg.MessageId,
		Value:     json.RawMessage(value),
	}
	complepeData, err := json.Marshal(complepeMsg)
	if err != nil {
		log.Fatalf("行为调用ask序列化失败，Err=%v", err)
		return
	}
	m.publish(task.TopicComplete(m.topicPrefix, msg.SenderClientId), complepeData, m.qos, m.retain)
}

func (m *MQTT) onTopicAsk(msg task.MessageAsk) {
	log.Printf("收到行为调用ask，MessageId=%s", msg.MessageId)
}
func (m *MQTT) onTopicComplete(msg task.MessageComplete) {
	log.Printf("收到行为调用Complete，MessageId=%s, Value=%v", msg.MessageId, msg.Value)
	if waiter, ok := m.waiters[msg.MessageId]; ok {
		waiter.Ch <- msg.Value
		delete(m.waiters, msg.MessageId)
	}
}
func (m *MQTT) onTopicFailed(msg task.MessageFailed) {
	log.Printf("收到行为调用Failed，MessageId=%s, Error=%v", msg.MessageId, msg.Error)
	if waiter, ok := m.waiters[msg.MessageId]; ok {
		waiter.Ch <- errors.New(string(msg.Error))
		delete(m.waiters, msg.MessageId)
	}
}
func (m *MQTT) SyncAction(action task.MessagePending) (*task.Waiter, error) {
	if action.MessageId == "" {
		return nil, errors.New("消息ID为空")
	}
	// 创建等待器
	waiter := task.NewWaiter(action.MessageId)
	m.waiters[action.MessageId] = waiter

	// 序列化消息为JSON
	jsonData, err := json.Marshal(action)
	if err != nil {
		delete(m.waiters, action.MessageId)
		return nil, err
	}

	err = m.publish(task.TopicPending(m.topicPrefix, action.ReceiverClientId), jsonData, m.qos, m.retain)
	if err != nil {
		delete(m.waiters, action.MessageId)
		return nil, err
	}

	return waiter, nil
}
func (m *MQTT) publish(topic string, payload []byte, qos byte, retain bool) error {
	token := m.paho.Publish(topic, qos, retain, payload)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTT) subscribe(topic string, qos byte, callback pahomqtt.MessageHandler) error {
	token := m.paho.Subscribe(topic, qos, callback)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTT) unsubscribe(topic string) error {
	token := m.paho.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
