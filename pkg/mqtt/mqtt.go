package mqtt

import (
	"errors"
	"fmt"
	"time"

	"encoding/json"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
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
	logger  zerolog.Logger
}

func NewMQTT(config types.MQTTClientOpts, logger zerolog.Logger) (*MQTT, error) {
	if config.Broker == "" {
		return nil, errors.New("mqtt config broker is empty")
	}

	m := &MQTT{
		config:             config,
		topicPrefix:        config.TopicPrefix,
		qos:                1,
		retain:             false,
		cleanSession:       false,
		subscribeActionArr: make(map[string]MessageHandler),
		waiters:            make(map[string]*task.Waiter),
		logger:             logger,
	}

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
		m.logger.Error().Msgf("MQTT连接断开: %v", err)
	})

	opts.SetOnConnectHandler(func(client pahomqtt.Client) {
		m.logger.Info().Msg("MQTT连接成功")
	})
	m.paho = pahomqtt.NewClient(opts)

	return m, nil
}

func (m *MQTT) Connect() error {

	connToken := m.paho.Connect()
	// 等待连接完成，设置超时
	if !connToken.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("MQTT连接超时，请检查服务器地址或网络连接：%s", m.config.Broker)
	}

	if connToken.Error() != nil {
		return fmt.Errorf("MQTT连接失败，请检查用户名密码是否正确：%s", connToken.Error())
	}

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
			m.logger.Error().Msgf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		// 这里我都不if判断了，如果别人往这个pending主题胡乱投递不符合task.MessagePending的数据结构，那么后续处理时候自然会报错的。
		m.onTopicPending(message)
	})
	m.subscribe(task.TopicAsk(m.topicPrefix, m.config.Username), m.qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		var message task.MessageAsk
		mqttMessage := msg.Payload()
		if err := json.Unmarshal(mqttMessage, &message); err != nil {
			m.logger.Error().Msgf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		m.onTopicAsk(message)
	})
	m.subscribe(task.TopicComplete(m.topicPrefix, m.config.Username), m.qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		var message task.MessageComplete
		mqttMessage := msg.Payload()
		if err := json.Unmarshal(mqttMessage, &message); err != nil {
			m.logger.Error().Msgf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		m.onTopicComplete(message)
	})
	m.subscribe(task.TopicFailed(m.topicPrefix, m.config.Username), m.qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		var message task.MessageFailed
		mqttMessage := msg.Payload()
		if err := json.Unmarshal(mqttMessage, &message); err != nil {
			m.logger.Error().Msgf("解析消息失败: err=%v, message=%s", err, mqttMessage)
			return
		}
		m.onTopicFailed(message)
	})
}
func (m *MQTT) onTopicPending(msg task.MessagePending) {
	// 如果已经超过时间，则丢弃
	if msg.Expiration < time.Now().Unix() {
		m.logger.Warn().Msgf("消息已过期，丢弃，messageId=%s", msg.MessageId)
		return
	}
	// 先回复一个ask
	askMsg := task.MessageAsk{
		MessageId: msg.MessageId,
	}
	askData, err := json.Marshal(askMsg)
	if err != nil {
		m.logger.Error().Msgf("行为调用ask序列化失败，Err=%v", err)
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
		m.logger.Error().Msgf("行为调用没有找到回调函数，actionName=%s", msg.Action)
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
			m.logger.Error().Msgf("行为调用ask序列化失败，Err=%v", err)
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
		m.logger.Error().Msgf("行为调用ask序列化失败，Err=%v", err)
		return
	}
	m.publish(task.TopicComplete(m.topicPrefix, msg.SenderClientId), complepeData, m.qos, m.retain)
}

func (m *MQTT) onTopicAsk(msg task.MessageAsk) {
}
func (m *MQTT) onTopicComplete(msg task.MessageComplete) {
	// 同步行为调用接收响应
	if waiter, ok := m.waiters[msg.MessageId]; ok {
		waiter.Ch <- msg.Value
		delete(m.waiters, msg.MessageId)
	}
}
func (m *MQTT) onTopicFailed(msg task.MessageFailed) {
	// 同步行为调用接收响应
	if waiter, ok := m.waiters[msg.MessageId]; ok {
		waiter.Ch <- errors.New(string(msg.Error))
		delete(m.waiters, msg.MessageId)
	}
}
func (m *MQTT) RsyncAction(action task.MessagePending) error {
	return m.action(action)
}
func (m *MQTT) SyncAction(action task.MessagePending) (*task.Waiter, error) {
	waiter := task.NewWaiter(action.MessageId, time.Unix(action.Expiration, 0))
	m.waiters[action.MessageId] = waiter
	err := m.action(action)
	if err != nil {
		delete(m.waiters, action.MessageId)
		return nil, err
	}
	return waiter, nil
}

func (m *MQTT) action(action task.MessagePending) error {
	if action.MessageId == "" {
		return errors.New("消息ID为空")
	}
	if action.Expiration == 0 {
		return errors.New("超时时间必须设置")
	}
	if action.Expiration > time.Now().Add(3*24*time.Hour).Unix() {
		return errors.New("超时时间不得大于3天")
	}

	// 序列化消息为JSON
	jsonData, err := json.Marshal(action)
	if err != nil {
		return err
	}

	err = m.publish(task.TopicPending(m.topicPrefix, action.ReceiverClientId), jsonData, m.qos, m.retain)
	if err != nil {
		return err
	}

	return nil
}
func (m *MQTT) Report(selfClientId string, status types.Status) error {
	statusJSON, err := json.Marshal(status)
	if err != nil {
		return err
	}
	err = m.publish(task.TopicStatus(m.topicPrefix, selfClientId), statusJSON, m.qos, true)
	if err != nil {
		return err
	}

	return nil
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
