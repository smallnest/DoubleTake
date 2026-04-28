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
- `game.go`: 游戏逻辑（预留）
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
