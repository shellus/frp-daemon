package main

import (
	"fmt"
	"log"
	"os"

	config "github.com/shellus/frp-daemon/pkg/client"
	"github.com/shellus/frp-daemon/pkg/types"
)

type Handler struct{}

// existsInstance 检查实例是否存在
func existsInstance(instances []types.InstanceConfigLocal, name string) bool {
	for _, instance := range instances {
		if instance.Name == name {
			return true
		}
	}
	return false
}

// removeInstance 移除实例
func removeInstance(instances []types.InstanceConfigLocal, name string) []types.InstanceConfigLocal {
	for i, instance := range instances {
		if instance.Name == name {
			return append(instances[:i], instances[i+1:]...)
		}
	}
	return instances
}

// HandleUpdate 处理下发frp实例
func (h *handler) HandleUpdate(instance types.InstanceConfigRemote) {
	log.Printf("处理下发frp实例指令: %s", instance.Name)

	// 检查已存在实例，停止并从内存配置中移除
	if existsInstance(instancesFile.Instances, instance.Name) {
		log.Printf("实例 %s 已存在，停止实例", instance.Name)
		if err := StopFrpInstance(instance.Name); err != nil {
			log.Printf("停止实例 %s 失败: %v", instance.Name, err)
			return
		}
		instancesFile.Instances = removeInstance(instancesFile.Instances, instance.Name)
		log.Printf("已存在的实例 %s 已停止", instance.Name)
	} else {
		log.Printf("实例 %s 不存在，开始创建", instance.Name)
	}

	// 配置写入本地文件得到文件名
	filePath := fmt.Sprintf("%s/%s.yaml", instancesDir, instance.Name)
	// 写入文件
	err := os.WriteFile(filePath, []byte(instance.ConfigContent), 0644)
	if err != nil {
		log.Printf("写入实例配置文件失败: %v", err)
	}

	// 生成本地实例配置
	localInstance := types.InstanceConfigLocal{
		Name:       instance.Name,
		Version:    instance.Version,
		ConfigPath: filePath,
	}
	instancesFile.Instances = append(instancesFile.Instances, localInstance)

	// 启动实例
	if err := StartFrpInstance(localInstance); err != nil {
		log.Printf("启动实例 %s 失败: %v", localInstance.Name, err)
	}

	// 更新实例配置文件
	err = config.WriteInstancesFile(instancesPath, instancesFile)
	if err != nil {
		log.Printf("写入实例配置文件失败: %v", err)
	}

	log.Printf("实例 %s 启动成功，已更新实例配置文件", localInstance.Name)
}
