package controller

import (
	"errors"

	"github.com/shellus/frp-daemon/pkg/types"
)

type Controller struct {
	auth *types.ClientAuth
	mqtt *types.MQTTClientOpts
}

func NewController(auth *types.ClientAuth, mqtt *types.MQTTClientOpts) (*Controller, error) {
	if auth == nil {
		return nil, errors.New("auth is nil")
	}

	if mqtt == nil {
		return nil, errors.New("mqtt is nil")
	}

	return &Controller{
		auth: auth,
		mqtt: mqtt,
	}, nil
}

// 连接MQTT
func (c *Controller) ConnectMQTT() error {

	return errors.New("not implemented")
}

// 实现配置下发
func (c *Controller) SendConfig(clientId string, config *types.InstanceConfigLocal) error {

	return errors.New("not implemented")
}

// 列出客户端实例
func (c *Controller) ListInstances(clientId string) ([]types.InstanceConfigLocal, error) {

	return nil, errors.New("not implemented")
}

// 删除客户端实例
func (c *Controller) DeleteInstance(clientId string, instanceName string) error {

	return errors.New("not implemented")
}

// 查看指定实例的lastLog
func (c *Controller) GetLastLog(clientId string, instanceName string) ([]string, error) {

	return nil, errors.New("not implemented")
}

// 查看指定实例的status
func (c *Controller) GetStatus(clientId string, instanceName string) ([]types.InstanceStatus, error) {

	return nil, errors.New("not implemented")
}

//
