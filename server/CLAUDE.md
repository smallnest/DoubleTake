# server

## 架构约定
- 包名与目录名一致，使用 `server` 包
- Server 结构体管理 TCP 监听和所有客户端连接
- Player 结构体代表一个已连接的客户端，包含 `Conn net.Conn` 和 `Name string` 字段
- 连接注册表使用 `map[net.Conn]*Player`，通过 `sync.Mutex` 保护并发访问
- 已注册名字集合使用 `map[string]bool`（`names` 字段），由同一个 `sync.Mutex` 保护
- 使用 `done` channel 实现 Stop() 的优雅关闭，Accept 循环通过 select 检查 done

## 依赖关系
- 依赖 `game` 包的 `Message` 类型、`Encode`/`Decode` 函数
- 依赖标准库: `net`, `bufio`, `log`, `sync`, `fmt`

## 消息处理流程
1. Start() 监听 TCP 端口，循环 Accept 连接
2. 每个连接启动独立 goroutine 运行 handleConn
3. handleConn 使用 bufio.Scanner 按行读取，调用 game.Decode 解析
4. 解析失败仅记录日志不关闭连接（允许客户端恢复）
5. 有效消息通过 switch msg.Type 路由到对应处理函数（如 handleJoin、handleDesc）
6. 客户端断开时（scanner.Scan() 返回 false），defer 中 unregister 清理连接和注册名字

## 并发安全模式
- `listener` 字段由 `sync.Mutex` 保护，Start() 中赋值和 Stop() 中读取都在锁内
- `ready` channel 确保 Stop() 等待 Start() 完成 listener 设置后再关闭，避免竞态
- `stopOnce` (`sync.Once`) 防止 Stop() 重复调用导致 `close(done)` panic
- Stop() 中先收集连接并清空 map（持锁），再在锁外逐个 Close，避免与 unregister() 死锁
- unregister() 也获取 `s.mu`，因此绝不能在持锁期间调用 conn.Close()

## 注意事项
- Start() 是阻塞方法，需在 goroutine 中调用
- Stop() 是幂等的（可安全多次调用）
- Stop() 会阻塞直到 Start() 完成 listener 绑定（通过 `<-s.ready`）
- Stop() 关闭 listener 后 Accept 会返回错误，通过 done channel 区分正常关闭与异常
- Broadcast 和 Send 使用 game.Encode 编码消息，格式为 `TYPE|payload\n`
- unregister 中会关闭连接的 Conn，不要重复 Close
- handleJoin 等消息处理函数中，先解锁再调用 Send()，避免在持锁期间执行网络 I/O（防止死锁）
- Player.Name 在连接期间有效，断开后 unregister 会从 names map 中移除
- SendToPlayer(name, msg) 按玩家名查找连接并私密发送消息，查不到时返回 error
- SendToPlayer 持锁期间执行 conn.Write（与 Broadcast 模式一致），因最大连接数 <= 10，阻塞风险可忽略
- 目前没有 name→conn 反向映射，SendToPlayer 使用 O(n) 遍历 connections map

## DESC 消息处理
- `handleDesc` 处理 DESC 消息：未命名玩家收到 ERROR("请先加入游戏")，已命名玩家的描述转发到 `OnDescMsg` channel
- `OnDescMsg` channel 缓冲大小 64，满时丢弃消息并记录日志
- `DescEvent` 结构体包含 `PlayerName` 和 `Description` 字段
- server 层不做描述内容校验（空描述、非当前发言者），这些由 judge 端 `descriptionPhase` 通过 `DescRound.RecordDesc` 处理

## VOTE / PK_VOTE 消息处理
- `handleVote` 处理 VOTE 消息，转发到 `OnVoteMsg` channel（`VoteEvent{PlayerName, Target}`）
- `handlePKVote` 处理 PK_VOTE 消息，转发到 `OnPKVoteMsg` channel（`VoteEvent{PlayerName, Target}`）
- 两个 handler 的模式与 `handleDesc` 完全一致：未命名玩家收到 ERROR，已命名玩家转发到 channel
- `OnVoteMsg` 和 `OnPKVoteMsg` channel 缓冲大小均为 64，满时丢弃并记录日志
- `VoteEvent` 结构体包含 `PlayerName` 和 `Target` 字段
- server 层不做投票合法性校验（空目标、非玩家目标、重复投票），这些由 judge 端通过 `VoteRound.RecordVote` 或 `PKRound.RecordPKVote` 处理

## QUIT 消息处理
- `handleQuit` 处理 QUIT 消息：未命名玩家直接返回（deferred unregister 清理连接），已命名玩家广播 `QUIT|playerName` 给其他已命名玩家
- `OnQuitMsg` channel 缓冲大小 64，满时丢弃事件（不记录日志，因为退出是低频事件）
- `QuitEvent` 结构体包含 `PlayerName` 字段
- handleConn 中收到 MsgQuit 后调用 handleQuit 并 return 退出读取循环，deferred unregister 负责从 names map 中移除并关闭连接
- 广播 QUIT 时排除发送者自身（`p.Conn == player.Conn`）和未命名玩家（`p.Name == ""`）
- handleQuit 持锁期间执行 conn.Write 广播（与 Broadcast 模式一致），因最大连接数 <= 10，阻塞风险可忽略
