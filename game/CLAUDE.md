# game

## 架构约定
- 包名与目录名一致，使用 `game` 包
- 协议常量使用 `Msg` 前缀 + 大写驼峰命名（如 `MsgPKStart`），避免与 Go 保留字冲突
- 常量定义集中放在 `protocol.go` 中，使用单个 `const` 块组织

## 目录结构
- `protocol.go`: 通信协议消息类型常量、Message 结构体、Encode/Decode 函数
- `protocol_test.go`: Encode/Decode 单元测试
- `game.go`: 游戏逻辑（预留）

## 协议编解码约定
- 线路格式: `TYPE|payload\n`
- Encode: 简单拼接 `Type + "|" + Payload + "\n"`
- Decode: 使用 `strings.SplitN(line, "|", 2)` 仅分割第一个 `|`，payload 可包含 `|`
- Decode 使用 `strings.TrimRight` 去除末尾换行符
- 空消息或仅换行符的消息返回 `ErrInvalidMessage`
- 缺少 `|` 分隔符的消息返回 `ErrInvalidMessage`
