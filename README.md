# 一个frp守护进程
基于go语言，通过MQTT协议实现一个无服务器的客户端和控制端，动态下发frp配置，让设备永远不会失联。

## 架构
- [cmd/fdctl, pkg/fdctl]：在主控端运行，负责生成客户端配置，{下发，删除，更新}frp实例，查看指定客户端指定frp实例状态。
- [cmd/fdclient, pkg/fdclient]：在被控端机器上运行，配置由主控端生成并储存索引，负责启动frp实例，管理frp实例，等待主控指令。
- [pkg/mqtt]：封装MQTT连接，发送，接收消息，主要作用是封装了一个点对点任务架构，允许业务层发送指令并得到结果。
- [pkg/emqx]：用于调用emqx的api，为被控端创建mqtt用户。
- [pkg/frp]：针对frp的进程管理器，用于启动停止frp实例。
- [pkg/installer]：用于安装指定版本的frp二进制文件。
- [pkg/types]：主控被控端共用的业务数据结构。

## 开始
- 获得emqx的api_app_key和api_secret_key，写入controller.yml
- 运行fdctl new -name fdctl，创建一组mqtt和client信息，写入controller.yml，因为控制端也是一个标准的client。

## 进度
- [✓] 下载指定版本frp二进制
- [✓] 运行多个frp实例
- [✓] 优雅关闭
- [✓] client连接MQTT等待配置下发
- [✓] `fdctl new -name <clientName>`用于创建一个新的客户端
- [✓] controller命令：ping
- [✓] controller命令：delete
- [✓] `fdctl update -name <clientName> -instance <instanceName> --version 0.51.2 -config temp/frpc.ini`用于更新指定实例的配置
- [ ] controller命令：status
- [ ] controller命令：tail
- [ ] 实现MQTT保留消息，允许客户端不在线也可以下发配置
