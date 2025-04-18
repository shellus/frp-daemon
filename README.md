# 一个frp守护进程
本程序通过MQTT，实现一个无服务器的客户端和控制端，动态下发frp配置，防止设备离线。

## 进度
- [✓] 下载指定版本frp二进制
- [✓] 运行多个frp实例
- [✓] 优雅关闭
- [✓] client连接MQTT等待配置下发
- [✓] controller命令：new
- [ ] controller命令：delete
- [ ] controller命令：update
- [ ] controller命令：status
- [ ] controller命令：tail
- [ ] 实现MQTT保留消息，允许客户端不在线也可以下发配置
