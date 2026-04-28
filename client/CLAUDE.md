# client

## 架构约定
- 包名与目录名一致，使用 `client` 包
- Client 结构体管理 TCP 连接和消息收发
- 使用 `messages` channel (buffered, 64) 向主循环传递接收到的消息
- 使用 `done` channel 和 `closeOnce` 实现 Disconnect() 的幂等优雅关闭

## 依赖关系
- 依赖 `game` 包的 `Message` 类型、`Encode`/`Decode` 函数
- 依赖标准库: `net`, `bufio`, `fmt`, `sync`

## 消息处理流程
1. Connect() 建立 TCP 连接并启动 receiveLoop goroutine
2. receiveLoop 使用 bufio.Scanner 按行读取，调用 game.Decode 解析
3. 解析失败的消息静默跳过（与服务端行为一致）
4. 有效消息通过 messages channel 传递给调用方
5. 收到 EOF 或错误时自动调用 Disconnect() 优雅退出

## 并发安全模式
- `closeOnce` (`sync.Once`) 防止 Disconnect() 重复调用导致 panic
- Disconnect() 关闭 done channel 和 conn，然后关闭 messages channel
- receiveLoop 通过 select 同时监听 messages 和 done，避免 Disconnect 后仍阻塞写入

## 注意事项
- Disconnect() 是幂等的（可安全多次调用）
- Messages() 返回只读 channel，调用方通过 range 或 select 读取
- 服务端关闭连接时，客户端通过 scanner 停止检测到 EOF 并自动 Disconnect
- Send() 在未连接时返回错误，不 panic
