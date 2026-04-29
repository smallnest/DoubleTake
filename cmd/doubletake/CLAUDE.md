# cmd/doubletake

## 架构约定
- CLI 入口点，二进制名称 `doubletake`
- 核心逻辑在 `run(stdout, stderr io.Writer, stdin io.Reader, args []string) int` 中，`main()` 只负责调用
- `run` 接受 `io.Writer` 和 `io.Reader` 参数，便于测试捕获输出和模拟输入
- 手动解析 flag（不使用 `flag` 包），避免 `flag.Parse()` 调用 `os.Exit` 导致测试无法捕获
- 通过 `client.NewDisplay(stdout, stealth)` 创建 Display 实例进行输出，不直接 `fmt.Fprintf`
- 角色分发使用 `switch role` 结构，`RunJudge` 和 `RunPlayer` 各自独立处理
- ROLE 消息加载格式：服务端发送 `ROLE|Civilian|苹果`（`roleName|word`），play 端 `splitN(msg.Payload, "|", 2)` 解析
- 玩家端收到 ROLE 后显示 `你的身份: 苹果 [平民]`，白板显示 `你的身份: [白板] — 你是白板`
- 使用 `roleDisplayNames` map 将英文角色名映射为中文显示标签
- 所有用户可见的输出信息使用中文

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
- `RunJudge` 流程：collectConfig → waitingPhase（首次）→ 会话循环（collectWordsFromCh → AssignRoles → AssignWords → broadcastReady → 游戏循环 → 询问新一局）
- 游戏结束后裁判被询问"是否开始新一局？(Y/N)"，选 Y 继续会话循环，选 N 退出
- 白板人数输入支持回车默认 0（`readIntInputOrDefault` 函数）
- 裁判创建房间时显示房间码（`game.EncodeRoomCode`）
- 集成测试中 `extraLines` channel 提供的输入不仅包括 "start"/"Y"，还需包括词语输入（如 "苹果"、"香蕉"）
- `descriptionPhase` 在 `broadcastReady` 之后被调用，广播 ROUND|轮次号|发言顺序 和 TURN|playerName 消息

## 测试集成验证模式
- 集成测试验证 ROLE 消息内容时，使用 `strings.SplitN(msg.Payload, "|", 2)` 解析 roleName 和 word，与服务端 `sendRoleToPlayers` 保持一致
- 验证角色分布使用计数方式（统计 Civilian/Undercover/Blank 出现次数），而非按具体玩家名匹配，因为 `AssignRoles` 使用 `rand.Shuffle` 随机分配
- 集成测试读取消息时必须 **按连接依次读取 ROLE + READY**（先读完一个连接的 ROLE+READY，再读下一个），否则服务端可能在读完所有 ROLE 后已关闭连接导致 READY 读取失败
- 白板玩家在协议层收到 `ROLE|Blank|你是白板`，玩家端显示 `你的身份: [白板] — 你是白板`
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

## 游戏循环约定
- `RunJudge` 使用外层会话循环 + 内层游戏循环管理多局多轮
- 每轮流程：`descriptionPhase` → `votingPhase` → `CheckWinCondition` → 未结束则更新 `alivePlayers` 进入下一轮
- `getAlivePlayers(players)` 辅助函数从 `[]*game.Player` 提取存活玩家名列表，每轮投票后重新计算
- 游戏结束时构造 WIN payload 广播给所有玩家，然后 break 内层循环，询问裁判是否开新局
- WIN payload 格式：`"winner|player1:Role:alive,player2:Role:alive,...|civilianWord|undercoverWord"`
  - `winner` 为 `"Civilian"` 或 `"Undercover"`
  - `alive` 为 `"1"`（存活）或 `"0"`（已淘汰）
  - `buildWinPayload(winner, players, civilianWord, undercoverWord)` 辅助函数构造 payload
- `CheckWinCondition` 在 `votingPhase` 返回后调用（而非在 `votingPhase` 内部），保持职责分离
- 获胜信息使用中文角色名（通过 `roleDisplayNames` 映射）

## 玩家端多会话约定
- 收到 WIN 消息后不退出，重置阶段状态（`descP=descIdle`, `voteP=voteIdle`, `inVotePhase=false`, `guessPending=false`）并 `return -1` 继续主循环
- 玩家保持连接，等待下一会话的 ROLE/READY 消息
- stdin goroutine 跨会话共享，不重复创建

## 断线超时跳过约定
- `disconnectTimeout` 为包级变量（`var`，非 `const`），默认 60 秒，测试中可临时覆盖为短时长
- `stopTimer(timer)` 安全停用 timer 并 drain 其 channel，防止 select 泄漏
- `isPlayerDisconnected(name, players, playersMu)` 检查 `game.Player.Connected` 字段，需持有 `playersMu` 锁
- `descriptionPhase` 和 `votingPhase` 的循环均使用 `select` 多路复用 `OnDescMsg`/`OnVoteMsg` 与 `timeoutCh`
- 仅当 `isPlayerDisconnected` 返回 true 时才启动 timer；已连接玩家走原始阻塞路径（timer=nil, timeoutCh=nil→select 忽略该 case）
- 掉线玩家在超时前重连并发送有效消息（DESC/VOTE），会被 `OnDescMsg`/`OnVoteMsg` 正常处理（`stopTimer` 取消超时）
- 超时后调用 `round.SkipCurrent()`/`v.SkipCurrent()` 并广播 `TURN` 给下一位
- `pkPhase` 的描述子循环和投票子循环同样遵循此模式
- 所有四个循环的 `select` 均包含 `case disc := <-srv.OnDisconnect:` 处理掉线事件：当当前操作者掉线时启动 timer，防止已连接玩家在轮到其操作后掉线导致游戏永久挂起

## 断线超时跳过测试约定
- 单元测试 `TestIsPlayerDisconnected` 验证 `isPlayerDisconnected` 函数逻辑（Connected=true→false, Connected=false→true, 未知玩家→false）
- 超时跳过集成测试使用 `setupDescPhaseTestFull`/`setupVotePhaseTestFull`（返回 out+port），通过 `disconnectTimeout` 包级变量覆盖为短时长（如 50ms）加速测试
- 测试超时跳过的标准模式：读取 ROUND+TURN/VOTE+TURN → 关闭当前操作者连接 → 从另一玩家发送无效消息（DESC/VOTE）触发 judge 循环重入 → 等待 `waitForOutput(t, out, "超时未发言（已掉线），跳过"...)` 确认超时触发 → 读取 TURN 确认下一位操作者
- 超时跳过后验证：剩余连接收到 TURN 消息，且 Payload 不再是已掉线玩家名
- 重连测试使用 `setupDescPhaseTestFull`/`setupVotePhaseTestFull`（返回 port），覆盖 `disconnectTimeout` 为较长时长（如 5s）以确保重连在超时前完成
- 重连测试标准模式：关闭连接 → 从另一玩家发送无效消息触发循环重入 → 通过 `net.Dial("tcp", port)` 建立新连接 → 发送 `RECONNECT|playerName` → 消费 RECONNECT+STATE 确认 → 发送有效 DESC/VOTE → 验证正常处理（DESC广播/TURN下一位） → 确认输出不包含 `"超时未发言"`/`"超时未投票"`
- 重连测试注意事项：重连后的投票目标不能是自己（避免 `ErrVoteSelf`）

## 游戏循环测试约定
- `setupGameLoopTest` 辅助函数处理完整初始化流程，返回 conns + speakers + cleanup
- `doDescription` 辅助函数驱动描述阶段（所有玩家依次发言）
- `doVotingForElimination` 辅助函数驱动投票阶段，指定目标玩家被投票出局
- 测试 WIN 消息时需处理不确定性：投票出局的玩家可能是卧底（游戏结束）或平民（游戏继续）
- `TestGameLoop_WinBroadcastOnCivilianWin` 根据实际结果分支验证 WIN 或 ROUND 消息
- `TestBuildWinPayload` 单元测试验证 payload 格式，使用固定的 players 切片避免随机性

## 投票环节约定
- `votingPhase` 严格复用 `descriptionPhase` 的模式：NewVoteRound → 广播 VOTE → TURN → 循环读 OnVoteMsg → RecordVote 校验 → ERROR 拒绝 → TURN 下一位 → Tally + FindEliminated → 广播 RESULT
- 广播消息格式：`VOTE|roundNum|alivePlayerList`（逗号分隔）、`RESULT|playerA:count,playerB:count,...`
- `RecordVote` 需要 `alivePlayers` 参数用于校验（区分"已出局"和"不存在"），从 `players` 切片的 `Alive` 字段构建
- 投票结束后调用 `FindEliminated()` 找出局者，设置 `p.Alive = false`
- 平票时 `FindEliminated` 返回空字符串和 `tie=true`，`votingPhase` 不标记任何人出局
- `voteResult` 结构体包含 `Round *game.VoteRound` 和 `Eliminated string`（空串表示平票或无人出局）

## 投票环节测试约定
- `setupVotePhaseTest` 辅助函数在 `setupDescPhaseTest` 基础上完成描述阶段（所有玩家发言），使连接进入投票阶段
- 投票阶段消息序列：VOTE → TURN（首位投票者）→ [TURN（下一位）] × N-1 → RESULT
- 测试中投票目标不能是自己：安排投票策略时必须确保投票者不投给自己（如 voters[0] 投 voters[1]，其他人投 voters[0]）
- 集成测试覆盖：正常投票流程、校验拒绝（空目标/投自己）、非当前投票者拒绝、结果广播

## 玩家端投票环节约定
- `player.go` 使用三态 `votePhase` 类型跟踪投票阶段状态：`voteIdle`（空闲/等待他人）、`voteWaitingInput`（等待 stdin 输入）、`voteSubmitted`（已提交等待服务端响应）
- `inVotePhase` 布尔标志用于区分当前处于描述阶段还是投票阶段——收到 `MsgVote` 消息时设置为 true，TURN 处理据此判断显示"请输入描述"还是"请输入投票目标"
- **关键**: `voteIdle` 既表示"尚未进入投票阶段"（`inVotePhase=false`）也表示"投票阶段中等待他人"（`inVotePhase=true`），所以必须用 `inVotePhase` 布尔标志来区分这两种情况
- 主循环的 `waitingInput` 条件检查 `descP == descWaitingInput || voteP == voteWaitingInput`，覆盖两个阶段的 stdin 需求
- stdin goroutine 在首次进入任一 waitingInput 状态时启动，跨阶段共享（不重复创建）
- VOTE 消息处理：解析 `roundNum|alivePlayerList`，显示"投票环节 轮次 N，可投票: P0 → P1"
- RESULT 消息处理：解析 `playerA:count,playerB:count,...`，逐行显示每人得票数
- ERROR 处理扩展：检查 `descP` 和 `voteP` 两个状态，在投票阶段重新提示"请输入投票目标"
- 客户端空投票拦截：stdin 读取到空行时本地拒绝（不发 VOTE），显示 "投票目标不能为空" 并重新提示
- `handleMessage` 签名为 `(msg, disp, out, cc, playerName string, descP *descPhase, voteP *votePhase, inVotePhase *bool, guessPending *bool)`

## PK 环节约定
- `pkPhase` 在 `votingPhase` 检测到平票后被调用，循环 PK 轮次直到分出唯一最高票
- PK 消息序列：PK_START → TURN（首位 PK 发言者）→ [DESC 广播 + TURN] × N（平票玩家描述）→ PK_VOTE → TURN（首位 PK 投票者）→ [TURN] × N-1 → RESULT
- PK 投票目标只能是平票玩家之一，不能投非平票玩家或自己
- PK 每轮若仍平票，`pkNum++` 并缩小平票范围后继续下一轮 PK
- 平票玩家列表来自 `FindTiedPlayers()`，顺序取决于 map 迭代（不确定），不等于描述阶段发言顺序
- PK 投票者顺序来自 `alivePlayers`（由 `players` 切片构建），与描述阶段发言顺序可能不同

## PK 环节测试约定
- **关键**: PK 投票顺序不等于描述发言顺序。`alivePlayers` 来自 `players` 切片（经 `AssignRoles` shuffle），所以 PK 投票者的实际顺序可能与 `voters`（描述发言顺序）不同
- 测试中必须通过 TURN 消息动态发现 PK 投票顺序，不能假设 `voters[0]` 就是第一个 PK 投票者
- `TestPKPhase_FullFlow` 中：先从 PK_VOTE 后的 TURN 消息获取 `firstPKVoter`，然后每次投票后读取 TURN 获取下一个投票者
- `TestPKPhase_InvalidVoteForNonTiedPlayer` 中：同样需要从 TURN 获取 `firstPKVoter`，而非使用 `voters[0]`
- PK 发言顺序来自 `NewPKRound` 中 `NewDescRound(pkNum, tied)`，tied 列表顺序来自 `FindTiedPlayers()`（map 迭代顺序不确定）
- `doVoting` 辅助函数用于驱动普通投票阶段，返回 RESULT 消息

## 玩家端投票环节测试约定
- 投票阶段测试复用 `startTestPlayerServer` 创建 TCP 服务器
- 测试覆盖：其他玩家投票回合显示"等待 X 投票..."、自己回合提示"请输入投票目标"并发送 VOTE|targetName、空目标客户端拦截、服务端 ERROR 重试、RESULT 结果显示、描述阶段→投票阶段过渡
- 描述→投票过渡测试验证：先发送 ROUND/TURN/DESC 消息完成描述阶段，再发送 VOTE/TURN 消息进入投票阶段，确认两个阶段的提示文案正确切换

## 玩家端 WIN 消息处理约定
- `handleMessage` 返回 `int`：`-1` 表示继续、`0` 表示正常退出（如 WIN）、`1` 表示错误退出（如致命 ERROR）
- `case game.MsgWin:` 调用 `handleWinMsg` 后重置阶段状态并 `return -1`（继续主循环，等待下一会话）
- `case game.MsgError:` 在非描述/投票阶段时 `return 1`（错误退出，exit code 1）
- `handleWinMsg` 解析 payload 格式 `"winner|playerStates|civilianWord|undercoverWord"`，使用 `SplitN(payload, "|", 4)` 分割
- playerStates 为逗号分隔的 `"name:Role:alive"` 列表，每个用 `SplitN(state, ":", 3)` 解析
- winner 和 roleName 都通过 `roleDisplayNames` 映射为中文显示标签（Civilian→平民, Undercover→卧底, Blank→白板）
- `handleWinMsg` 调用 `disp.ShowGameResult(winnerLabel, results, civilianWord, undercoverWord)` 显示结果
- WIN 消息处理测试覆盖：平民胜、卧底胜、stealth 模式
