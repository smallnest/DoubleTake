# cmd/doubletake

## 架构约定
- CLI 入口点，二进制名称 `doubletake`
- 核心逻辑在 `run(stdout, stderr io.Writer, stdin io.Reader, args []string) int` 中，`main()` 只负责调用
- `run` 接受 `io.Writer` 和 `io.Reader` 参数，便于测试捕获输出和模拟输入
- 手动解析 flag（不使用 `flag` 包），避免 `flag.Parse()` 调用 `os.Exit` 导致测试无法捕获
- 通过 `client.NewDisplay(stdout, stealth)` 创建 Display 实例进行输出，不直接 `fmt.Fprintf`
- 角色分发使用 `switch role` 结构，`RunJudge` 和 `RunPlayer` 各自独立处理
- ROLE 消息加载格式：服务端发送 `ROLE|Civilian|苹果`（`roleName|word`），play 端 `splitN(msg.Payload, "|", 2)` 解析
- 玩家端收到 ROLE 后以伪装格式显示 `assigned token: 苹果 [平民]`，白板显示 `assigned token: [白板] — 你是白板`
- 使用 `roleDisplayNames` map 将英文角色名映射为中文显示标签

## 命令行参数
- `--role string` — judge 或 player（必填）
- `--port int` — 裁判模式服务端口（默认 8127），玩家模式不使用此参数
- `--stealth` — 启用伪装精简输出模式（布尔开关，无值）
- `--help / -h` — 显示帮助

## 角色函数约定
- `RunJudge(out io.Writer, in io.Reader, port string, stealth bool)` — 裁判交互式配置流程，阻塞直到输入有效配置
- `RunPlayer(out io.Writer, in io.Reader, stealth bool)` — 玩家连接流程，读取房间码 → 解码 → 连接 → 输入名字 → 加入消息循环，阻塞直到服务端断开连接
- 两个函数都返回 `int`（退出码），通过 `bufio.NewScanner(in)` 读取输入（不使用 `Display.Prompt()` 因其硬编码 `os.Stdin`）

## 测试约定
- 测试文件 `main_test.go` 使用 `bytes.Buffer` 捕获 stdout/stderr
- 测试调用 `run()` 时必须传递 `strings.NewReader(input)` 作为 stdin 参数
- Judge 测试提供有效配置输入（如 `"6\n1\n0\n"`）
- Player 测试可使用空 `strings.NewReader("")` 触发快速失败，或设置真实 TCP 服务器端到端测试
- 覆盖场景：正常参数、无效 role、默认 port、自定义 port、缺失值、未知选项、help、stealth 开关
- 等待阶段集成测试使用 `net.Pipe()` 模拟 stdin，通过 channel 注入交互式输入（如 "start"、"Y"）
- 集成测试 cleanup 时必须关闭 extraLines channel，否则 stdin goroutine 会阻塞导致测试挂起
- 测试读取玩家连接消息时，需先消费 JOIN 确认消息，再检查后续的 READY 广播

## 注意事项
- `waitingPhase` 使用 `stdinSource`（ch + done channel）读取 stdin，由 `newStdinSource` 创建的 goroutine 持续读取 scanner 并发送到 channel
- `collectWordsFromCh` 也使用同一个 `stdinSource` channel，确保 waitingPhase 返回后词语输入不会被 stdin goroutine 抢先消费
- 嵌套的 `select`（如确认 Y/N 时）也必须包含 `<-stdinSrc.done` case，防止 EOF 时死锁
- `collectConfig` 中 `readIntInput` 在 EOF 时返回 -1 依赖 `validateConfig` 拒绝，会导致 EOF 场景下无限循环。如需优雅退出，应在 `collectConfig` 层检测 `scanner.Err()` 或 `readInt` 的 error
- `RunJudge` 流程：collectConfig → waitingPhase → collectWordsFromCh → AssignRoles → AssignWords → broadcastReady → descriptionPhase
- 集成测试中 `extraLines` channel 提供的输入不仅包括 "start"/"Y"，还需包括词语输入（如 "苹果"、"香蕉"）
- `descriptionPhase` 在 `broadcastReady` 之后被调用，广播 ROUND|轮次号|发言顺序 和 TURN|playerName 消息

## 测试集成验证模式
- 集成测试验证 ROLE 消息内容时，使用 `strings.SplitN(msg.Payload, "|", 2)` 解析 roleName 和 word，与服务端 `sendRoleToPlayers` 保持一致
- 验证角色分布使用计数方式（统计 Civilian/Undercover/Blank 出现次数），而非按具体玩家名匹配，因为 `AssignRoles` 使用 `rand.Shuffle` 随机分配
- 集成测试读取消息时必须 **按连接依次读取 ROLE + READY**（先读完一个连接的 ROLE+READY，再读下一个），否则服务端可能在读完所有 ROLE 后已关闭连接导致 READY 读取失败
- 白板玩家在协议层收到 `ROLE|Blank|你是白板`，玩家端显示 `assigned token: [白板] — 你是白板`
- Player 单元测试中服务端模拟发送多条消息时，不需要 `time.Sleep` 分隔发送，因为：
  1. 客户端 `messages` channel 缓冲大小为 64
  2. TCP 保证消息按序到达
  3. `bufio.Scanner` 正确按换行符分割消息

## 描述环节测试约定
- `readMsgFromConn` 使用 `readerForConn` 获取或创建 per-connection 的 `bufio.Reader`，**不能**每次调用都新建 `bufio.NewReader`，否则缓冲区中未读数据会丢失
- `readerForConn` 使用包级 `readers` map + `readersMu` mutex 管理连接到 Reader 的映射
- `setupDescPhaseTest` 辅助函数处理完整的游戏初始化流程：创建服务器 → 玩家加入 → start → 词语输入 → 消费 ROLE + READY
- 描述阶段的消息序列：ROUND → TURN（首位发言者）→ [DESC广播 + TURN（下一位）] × N-1 → DESC广播（最后一位）
- 集成测试读取描述阶段消息时，必须**分开读取 DESC 和 TURN**：先读所有连接的 DESC，再读所有连接的 TURN（因为服务器广播 DESC 后立即广播 TURN）
- `descriptionPhase` 使用 `game.DescRound` 管理发言顺序和状态，通过 `srv.OnDescMsg` channel 接收玩家描述
- 空描述和非当前发言者错误由 `descriptionPhase`（而非 server 层）处理，server 层只负责转发 DESC 消息

## 玩家端描述环节约定
- `player.go` 使用三态 `descPhase` 类型跟踪描述阶段状态：`descIdle`（空闲/等待他人）、`descWaitingInput`（等待 stdin 输入）、`descSubmitted`（已提交等待服务端响应）
- `descWaitingInput` 时使用 `select` 同时监听网络消息和 stdin 输入（通过 lazily initialized goroutine + channel）
- stdin goroutine 在首次进入 `descWaitingInput` 时启动，从同一个 `bufio.Scanner(in)` 读取后续行
- **关键**: 发送 DESC 后不应立即将 phase 设为 idle（`descSubmitted`），必须等服务端响应（DESC 广播表示接受，ERROR 表示拒绝需重新提示）
- ERROR 处理：`descSubmitted` 和 `descWaitingInput` 状态下重新提示输入，其他状态下直接退出
- TURN 消息区分自己/他人：比较 `msg.Payload` 与 `playerName`，自己时 prompt 输入，他人时显示等待提示
- 客户端空描述检查：stdin 读取到空行时本地拒绝（不发 DESC），显示 "描述不能为空" 并重新提示
- 消息格式化：ROUND→"轮次 N，发言顺序: P0 → P1 → P2"，DESC→"P0: 描述内容"，TURN(他人)→"等待 P0 描述..."

## 玩家端描述环节测试约定
- 描述阶段测试使用 `startTestPlayerServer` 创建 TCP 服务器，模拟发送 ROUND/TURN/DESC/ERROR 消息
- 测试覆盖：其他玩家回合显示等待提示、自己回合提示输入并发送 DESC、空描述客户端拦截、服务端 ERROR 重试、非描述阶段 ERROR 致命退出
- 服务端模拟 ERROR 重试测试：先读取第一个 DESC（被拒绝），发送 ERROR，再读取第二个有效 DESC
- 客户端空描述拦截测试：发送空行后客户端本地拦截，服务器只收到有效的描述
