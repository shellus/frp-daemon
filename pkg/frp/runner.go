package frp

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/shellus/frp-daemon/pkg/types"
)

// Runner FRP运行器
type Runner struct {
	instances map[string]*Instance
	mu        sync.RWMutex
}

// Instance FRP实例
type Instance struct {
	Name   string
	Config string
	cmd    *exec.Cmd
	status types.InstanceStatus
}

// NewRunner 创建FRP运行器
func NewRunner() *Runner {
	return &Runner{
		instances: make(map[string]*Instance),
	}
}

// StartInstance 启动FRP实例
func (r *Runner) StartInstance(name, version, frpPath, configPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查实例是否已存在
	if instance, exists := r.instances[name]; exists {
		if instance.cmd != nil && instance.cmd.Process != nil {
			return fmt.Errorf("实例 %s 已在运行", name)
		}
	}

	// 启动FRP实例
	cmd := exec.Command(frpPath, "-c", configPath)

	// TODO 日志应该记录在内存，只保留最后100行
	// cmd.Stdout = logFile
	// cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动FRP实例失败: %v", err)
	}

	// 保存实例信息
	r.instances[name] = &Instance{
		Name:   name,
		Config: configPath,
		cmd:    cmd,
		status: types.InstanceStatus{
			Running: true,
		},
	}

	// 监控实例状态
	go r.monitorInstance(name)

	return nil
}

// StopInstance 停止FRP实例
func (r *Runner) StopInstance(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, exists := r.instances[name]
	if !exists {
		return fmt.Errorf("实例 %s 不存在", name)
	}

	if instance.cmd != nil && instance.cmd.Process != nil {
		if err := instance.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("停止FRP实例失败: %v", err)
		}
		instance.cmd.Wait()
	}

	delete(r.instances, name)
	return nil
}

// GetStatus 获取实例状态
func (r *Runner) GetStatus() []types.InstanceStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var status []types.InstanceStatus
	for _, instance := range r.instances {
		status = append(status, instance.status)
	}
	return status
}

// monitorInstance 监控实例状态
func (r *Runner) monitorInstance(name string) {
	instance := r.instances[name]
	if instance == nil {
		return
	}

	err := instance.cmd.Wait()
	if err != nil {
		r.mu.Lock()
		instance.status.Running = false
		r.mu.Unlock()
	}

}
