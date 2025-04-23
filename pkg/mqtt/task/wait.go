package task

import (
	"errors"
	"time"
)

// 同步下发一个行为后，立即等待响应
type Waiter struct {
	messageId  string
	Ch         chan interface{}
	expiration time.Time
}

// Wait 等待任务完成或失败
func (w *Waiter) Wait() (value []byte, err error) {
	// 否则使用超时等待
	select {
	case result := <-w.Ch:
		switch v := result.(type) {
		case []byte:
			return v, nil
		case error:
			return nil, v
		default:
			return nil, nil
		}
	case <-time.After(time.Until(w.expiration)):
		return nil, errors.New("timeout")
	}
}

// NewWaiter 创建一个新的等待器
func NewWaiter(messageId string, expiration time.Time) *Waiter {
	return &Waiter{
		messageId:  messageId,
		Ch:         make(chan interface{}, 1),
		expiration: expiration,
	}
}
