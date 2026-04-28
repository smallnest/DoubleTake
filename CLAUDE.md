# DoubleTake

## 架构约定
- 模块路径: `github.com/smallnest/doubletake`
- Go 版本: 1.26.2
- 入口点: `cmd/doubletake/main.go`
- 包结构: `cmd/`(CLI 入口), `server/`(裁判模式服务端), `client/`(玩家模式客户端), `game/`(游戏逻辑)

## 依赖关系
- 各包保持独立，暂无跨包依赖

## 注意事项
- 二进制名称: `doubletake`，通过 `go build -o doubletake ./cmd/doubletake/` 构建
