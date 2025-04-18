package frp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

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
	logs   []string
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

	log.Printf("正在启动实例 %s，版本 %s，FRP路径 %s，配置路径 %s", name, version, frpPath, configPath)

	// 检查实例是否已存在
	if instance, exists := r.instances[name]; exists {
		if instance.cmd != nil && instance.cmd.Process != nil {
			return fmt.Errorf("实例 %s 已在运行", name)
		}
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件 %s 不存在", configPath)
	}

	// 启动FRP实例
	cmd := exec.Command(frpPath, "-c", configPath)

	// 创建管道用于捕获输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建标准输出管道失败: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建标准错误管道失败: %v", err)
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动FRP实例失败: %v", err)
	}

	// 保存实例信息
	instance := &Instance{
		Name:   name,
		Config: configPath,
		cmd:    cmd,
		status: types.InstanceStatus{
			Running: true,
			Pid:     cmd.Process.Pid,
			LastLog: make([]string, 0, 100),
		},
		logs: make([]string, 0, 100),
	}
	r.instances[name] = instance

	// 启动日志收集
	go r.collectLogs(instance, stdout, stderr)

	// 监控实例状态
	go r.monitorInstance(name)

	return nil
}

// collectLogs 收集实例日志
func (r *Runner) collectLogs(instance *Instance, stdout, stderr io.ReadCloser) {
	// 创建扫描器
	stdoutScanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)

	// 启动goroutine处理标准输出
	go func() {
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			fmt.Printf("[%s] %s\n", instance.Name, line)
			r.appendLog(instance, line)
		}
	}()

	// 启动goroutine处理标准错误
	go func() {
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			fmt.Printf("[%s] [ERROR] %s\n", instance.Name, line)
			r.appendLog(instance, line)
		}
	}()
}

// appendLog 添加日志行
func (r *Runner) appendLog(instance *Instance, line string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 添加到日志数组
	instance.logs = append(instance.logs, line)
	instance.status.LastLog = append(instance.status.LastLog, line)

	// 保持最后100行
	if len(instance.logs) > 100 {
		instance.logs = instance.logs[len(instance.logs)-100:]
		instance.status.LastLog = instance.status.LastLog[len(instance.status.LastLog)-100:]
	}
}

// StopInstance 停止FRP实例
func (r *Runner) StopInstance(name string) error {
	r.mu.Lock()
	instance, exists := r.instances[name]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("实例 %s 不存在", name)
	}

	// 标记实例为已停止
	instance.status.Running = false
	r.mu.Unlock()

	// 停止进程
	if instance.cmd != nil && instance.cmd.Process != nil {
		// 检查进程是否还在运行
		if err := instance.cmd.Process.Signal(syscall.Signal(0)); err == nil {
			// 进程还在运行，尝试终止它
			if err := instance.cmd.Process.Kill(); err != nil {
				log.Printf("警告: 终止进程失败: %v", err)
			}
			// 等待进程退出
			instance.cmd.Wait()
		} else {
			log.Printf("进程已经退出")
		}
	}

	// 删除实例
	r.mu.Lock()
	delete(r.instances, name)
	r.mu.Unlock()

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
	r.mu.RLock()
	instance := r.instances[name]
	r.mu.RUnlock()

	if instance == nil {
		return
	}

	err := instance.cmd.Wait()

	r.mu.Lock()
	defer r.mu.Unlock()

	// 再次检查实例是否存在，因为可能在等待过程中被删除
	if instance, exists := r.instances[name]; exists {
		instance.status.Running = false
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				instance.status.ExitStatus = exitErr.ExitCode()
			} else {
				instance.status.ExitStatus = -1
			}
		} else {
			instance.status.ExitStatus = 0
		}
	}
}

// Close 优雅关闭所有FRP实例
func (r *Runner) Close() error {
	r.mu.Lock()
	// 获取所有实例名称
	names := make([]string, 0, len(r.instances))
	for name := range r.instances {
		names = append(names, name)
	}
	r.mu.Unlock()

	var wg sync.WaitGroup
	var errs []error
	var errsMu sync.Mutex

	// 停止所有实例
	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := r.StopInstance(name); err != nil {
				errsMu.Lock()
				errs = append(errs, fmt.Errorf("停止实例 %s 失败: %v", name, err))
				errsMu.Unlock()
			}
		}(name)
	}

	// 等待所有实例停止完成
	wg.Wait()

	// 返回错误信息
	if len(errs) > 0 {
		return fmt.Errorf("关闭FRP实例时发生错误: %v", errs)
	}
	return nil
}

// GetInstancePid 获取实例的进程ID
func (r *Runner) GetInstancePid(name string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if instance, exists := r.instances[name]; exists {
		return instance.status.Pid
	}
	return 0
}
