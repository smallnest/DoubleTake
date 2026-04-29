# game

## 架构约定
- 包名与目录名一致，使用 `game` 包
- 协议常量使用 `Msg` 前缀 + 大写驼峰命名（如 `MsgPKStart`），避免与 Go 保留字冲突
- 常量定义集中放在 `protocol.go` 中，使用单个 `const` 块组织

## 目录结构
- `protocol.go`: 通信协议消息类型常量、Message 结构体、Encode/Decode 函数
- `protocol_test.go`: Encode/Decode 单元测试
- `base62.go`: 房间密码编解码（EncodeRoomCode/DecodeRoomCode），将 IP:port 编码为 base62 短字符串
- `base62_test.go`: 房间密码编解码单元测试
- `game.go`: 描述环节状态管理（DescRound 结构体及相关方法）
- `vote.go`: 投票环节状态管理（VoteRound 结构体及相关方法）
- `pk.go`: PK 环节状态管理（PKRound 结构体及相关方法），平票时触发
- `network.go`: 网络工具函数（GetLocalIP），获取本机非 loopback IPv4 地址
- `network_test.go`: 网络工具函数单元测试
- `role.go`: 角色定义（Role 类型、Player 结构体）和角色分配（AssignRoles 函数）
- `role_test.go`: 角色相关单元测试

## 命名约定
- `protocol.go` 中已有 `Encode`/`Decode` 函数用于消息编解码，房间密码编解码使用 `EncodeRoomCode`/`DecodeRoomCode` 以避免命名冲突

## 协议编解码约定
- 线路格式: `TYPE|payload\n`
- Encode: 简单拼接 `Type + "|" + Payload + "\n"`
- Decode: 使用 `strings.SplitN(line, "|", 2)` 仅分割第一个 `|`，payload 可包含 `|`
- Decode 使用 `strings.TrimRight` 去除末尾换行符
- 空消息或仅换行符的消息返回 `ErrInvalidMessage`
- 缺少 `|` 分隔符的消息返回 `ErrInvalidMessage`

## 网络工具函数约定
- `GetLocalIP()` 为纯工具函数，遍历 `net.Interfaces()` 取第一个 UP 且非 loopback 的 IPv4 地址
- 过滤 link-local 地址（169.254.x.x）
- 测试策略：不 mock `net.Interfaces`，通过验证返回值格式（IPv4 正则 + `net.ParseIP`）确保正确性

## 角色分配约定
- `Role` 类型使用 `int` + `iota`，三种角色：Civilian(0)、Undercover(1)、Blank(2)
- `Role` 实现了 `fmt.Stringer` 接口
- `AssignRoles` 使用 `math/rand.Shuffle` 随机打乱玩家顺序
- 参数校验顺序：空名字列表 -> 空字符串名字 -> 负数参数 -> 卧底+白板>=平民
- 错误信息使用 `errors.New` 或 `fmt.Errorf`，返回具体描述
- `Player` 初始状态：`Alive=true`，`Connected=false`
- 校验条件：`numUndercover+numBlank >= numCivilian`（即 U+B >= C），而非 `numCivilian <= 0`（后者只拦截了 U+B >= total 的极端情况）
- `AssignRoles` 不设置 `Word` 字段（留空），该字段在后续阶段设置
- `AssignWords` 在 `AssignRoles` 之后调用，根据角色设置每个 `Player` 的 `Word` 字段：Civilian→平民词语，Undercover→卧底词语，Blank→空字符串

## 测试约定
- `role_test.go` 使用表格驱动测试（`TestAssignRoles_Errors`）覆盖所有错误路径
- `wantErrSubstr` 字段用于验证错误消息包含预期子串，通过 `strings.Contains` 检查
- 成功路径测试按场景拆分为独立函数：`TestAssignRoles_Success`（5人）、`TestAssignRoles_FourPlayers`（4人1U0B）、`TestAssignRoles_TenPlayers`（10人3U1B）、`TestAssignRoles_AllCivilianBoundary`（4人全平民）
- 边界测试「刚好触发」场景（如 P=4, U=1, B=1, C=2, U+B=2 >= C）单独在 Error 表中覆盖
- 随机性测试 `TestAssignRoles_Shuffled` 使用 20 人 x 5 次试验，统计判断是否打乱
- Player 字段验证：每个成功路径测试均检查 `Alive=true`、`Connected=false`、`Name` 非空

## 描述环节约定
- `DescRound` 管理单轮描述阶段的状态：轮次号、发言顺序、当前发言者索引、描述记录 map
- `NewDescRound(roundNum int, alivePlayers []string)` 拷贝输入切片，避免共享底层数组
- `CurrentSpeaker()` 返回当前应发言的玩家名，全部发言完毕返回空字符串
- `RecordDesc(playerName, desc string)` 校验顺序：先检查空描述（`ErrEmptyDesc`），再检查是否轮到该玩家（`ErrNotYourTurn`）
- 空描述判定使用 `strings.TrimSpace`，纯空白视为空；有实际内容的描述保留原始内容（不做 trim）
- `AllDone()` 在 `CurrentIndex >= len(SpeakerOrder)` 时返回 true，0 人场景天然为 true
- 错误变量 `ErrEmptyDesc`、`ErrNotYourTurn` 定义在 `game.go` 中作为包级变量
- 描述记录使用 `map[string]string`，以玩家名为 key，方便裁判端按名回溯
- `game_test.go` 覆盖：正常流程、空描述拒绝（表格驱动）、非当前玩家拒绝、边界（0人、1人）、完整记录验证

## 投票环节约定
- `VoteRound` 管理单轮投票阶段的状态：轮次号、投票者列表、当前投票者索引、投票记录 map(voter→target)
- `NewVoteRound(roundNum int, alivePlayers []string)` 拷贝输入切片，避免共享底层数组；校验空名和重复名
- `CurrentVoter()` 返回当前应投票的玩家名，全部投票完毕返回空字符串
- `RecordVote(voter, target string, alivePlayers []string)` 校验顺序：空目标（`ErrVoteEmpty`）→ 非当前投票者（`ErrNotYourTurn`）→ 投自己（`ErrVoteSelf`）→ 投已出局玩家（`ErrVoteEliminated`）→ 投不存在的玩家（`ErrVoteUnknown`）
- 区分「已出局」与「不存在」的逻辑：先检查 target 是否在 alivePlayers 中，不在则检查 target 是否在 Voters（本局参与者）中——在 Voters 中说明是已出局玩家，不在则是完全不存在的玩家
- `Tally()` 返回 `map[string]int` 每个被投票者的得票数，仅统计被投的人
- `FindEliminated()` 返回票数最高者；平票返回空字符串和 `tie=true`；无投票返回空字符串和 `tie=false`
- 错误变量 `ErrVoteSelf`、`ErrVoteEliminated`、`ErrVoteUnknown`、`ErrVoteEmpty` 定义在 `vote.go` 中；`ErrNotYourTurn` 定义在 `game.go` 中为 DescRound 和 VoteRound 共用
- `vote_test.go` 覆盖：构造函数校验（空名/重复名）、正常投票流程、所有校验错误路径、平票场景、边界（0人、1人）、完整轮次集成测试
- 注意避免与 `game_test.go` 中已有测试函数同名（如 `TestAllDone_OnePlayer` 需加后缀 `_VoteRound`）

## PK 环节约定
- `PKRound` 管理平票 PK 阶段的状态：PK 轮次号、平票玩家列表、描述阶段（`DescRound`）、投票阶段
- `NewPKRound(pkNum int, tied []string, alivePlayers []string)` 校验：至少 2 人平票、无空名、无重复、平票玩家必须存活
- PK 分两个阶段：描述阶段（`Phase="desc"`，仅平票玩家发言）→ 投票阶段（`Phase="vote"`，所有存活玩家投票）
- 描述阶段复用 `DescRound`，发言者列表为平票玩家（非全部存活玩家）
- 投票阶段投票目标只能是平票玩家之一（`ErrVoteNotTied`），不能投非平票玩家或自己
- `TiedSet`（`map[string]bool`）用于快速校验投票目标是否为平票玩家
- `FindEliminated()` 仅统计平票玩家的得票，返回得票最高者；仍平票返回空字符串和 `tie=true`
- `pk_test.go` 覆盖：构造函数校验、描述阶段流程、投票阶段流程（含各类校验错误）、计票和平票判定、完整 PK 集成测试
