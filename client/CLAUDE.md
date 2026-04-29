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

## 已知问题与修复指南
- **并发 Disconnect 风险**: 当前 `Disconnect()` 在 `closeOnce.Do` 中 `close(c.messages)`，与 `receiveLoop` 的 `select { case c.messages <- msg }` 并发时可能 panic。修复方式：将 `close(c.messages)` 移到 `receiveLoop` 的 `defer` 中，`Disconnect()` 只负责 `close(c.done)` 和 `c.conn.Close()`。
- **scanner.Err()**: `receiveLoop` 中 scanner 退出后应调用 `scanner.Err()` 检查错误（与服务端约定一致），当前被忽略。

## 伪装显示模块 (display.go)
- `Display` 结构体接受 `io.Writer`，默认 `os.Stdout`，便于测试
- `stealth` 模式隐藏 `[DATA]`/`[INFO]`/`[WARN]` 标识，只保留 `[session:]`/`[node-]`/`warning:` 前缀
- `Prompt()` 从 `os.Stdin` 读取（不经过 `io.Writer`），因为需要真实终端交互
- 所有输出函数使用 `fmt.Fprintf(d.out, ...)` 写入，不直接 `fmt.Println`
- 所有带标识前缀的输出函数（Info, Warn, Data）都根据 stealth 字段切换输出格式
- `PlayerResult` 结构体（Name, Role, Alive）用于 `ShowGameResult` 的参数
- `ShowGameResult(winnerLabel, results, civilianWord, undercoverWord)` 以伪装格式显示游戏结果：标题行、每个玩家状态行、双方词语行
- `ShowGameResult` 测试覆盖正常模式和 stealth 模式

## 测试约定
- 使用 `startTestServer` 辅助函数创建临时 TCP 服务器
- 所有异步测试带 timeout 保护（1秒）
- 覆盖场景：连接成功/失败、发送/接收、断连、多次消息、无效消息
- Display 测试使用 `bytes.Buffer` 捕获输出，验证格式字符串精确匹配
- 每个 stealth 模式函数都有对应的独立测试用例（TestInfoStealth, TestWarnStealth, TestDataStealth）
- `ListPlayers` 使用 `fmt.Sprintf("[%d]", idx)` 而非 `string(rune('0'+idx))` 避免 idx>=10 时生成非数字字符
