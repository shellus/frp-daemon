package task

// 同步下发一个行为后，立即等待响应
type Waiter struct {
	MessageId string
	Ch        chan interface{}
}

// Wait 等待任务完成或失败
func (w *Waiter) Wait() (value []byte, err error) {
	result := <-w.Ch
	switch v := result.(type) {
	case []byte:
		return v, nil
	case error:
		return nil, v
	default:
		return nil, nil
	}
}

// NewWaiter 创建一个新的等待器
func NewWaiter(messageId string) *Waiter {
	return &Waiter{
		MessageId: messageId,
		Ch:        make(chan interface{}, 1),
	}
}
