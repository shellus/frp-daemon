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
	"time"

	"github.com/shellus/frp-daemon/pkg/types"
)

// Runner FRP运行器
type Runner struct {
	instances map[string]*Instance
	mu        sync.RWMutex
}

// Instance FRP实例本地配置
type Instance struct {
	Name       string
	ConfigPath string
	cmd        *exec.Cmd
	status     types.InstanceStatus
	logs       []string
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

	log.Printf("正在启动实例，instanceName=%s, version=%s, frpPath=%s, configPath=%s", name, version, frpPath, configPath)

	// 检查实例是否已存在
	if instance, exists := r.instances[name]; exists {
		if instance.cmd != nil && instance.cmd.Process != nil {
			return fmt.Errorf("实例已在运行，instanceName=%s", name)
		}
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在，configPath=%s", configPath)
	}

	// 启动FRP实例
	cmd := exec.Command(frpPath, "-c", configPath)

	// 创建管道用于捕获输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建标准输出管道失败，Error=%v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建标准错误管道失败，Error=%v", err)
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动FRP实例失败，Error=%v", err)
	}

	// 保存实例信息
	instance := &Instance{
		Name:       name,
		ConfigPath: configPath,
		cmd:        cmd,
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

// ExistsInstance 检查实例是否存在
func (r *Runner) ExistsInstance(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.instances[name]
	return exists
}

// StopInstance 停止FRP实例
func (r *Runner) StopInstance(name string) error {
	r.mu.Lock()
	instance, exists := r.instances[name]
	if !exists {
		r.mu.Unlock()
		// 实例不存在，视为成功
		return nil
	}

	// 标记实例为已停止
	instance.status.Running = false
	r.mu.Unlock()

	// 停止进程
	if instance.cmd != nil && instance.cmd.Process != nil {
		// 首先尝试发送SIGTERM信号，让进程优雅退出
		if err := instance.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// 如果进程已经退出，忽略错误
			if err.Error() != "os: process already finished" {
				log.Printf("警告: 发送SIGTERM信号失败: %v", err)
			}
		}

		// 等待进程退出，带超时
		time.Sleep(time.Second)               // 等待1秒
		if instance.cmd.ProcessState == nil { // 检查进程是否已退出
			log.Printf("警告: 进程 %d 未在1秒内退出，将在30秒后强制终止", instance.cmd.Process.Pid)
			time.Sleep(30 * time.Second)          // 等待30秒
			if instance.cmd.ProcessState == nil { // 再次检查进程是否已退出
				log.Printf("错误: 进程 %d 未在30秒内退出，发送SIGKILL信号强制终止", instance.cmd.Process.Pid)
				if err := instance.cmd.Process.Kill(); err != nil {
					log.Printf("警告: 发送SIGKILL信号失败: %v", err)
				}
			}
		}
	}

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

	// 等待进程退出
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
		// 删除实例
		delete(r.instances, name)
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
