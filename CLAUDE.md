# DoubleTake

## 架构约定
- 模块路径: `github.com/smallnest/doubletake`
- Go 版本: 1.26.2
- 入口点: `cmd/doubletake/main.go`
- 包结构: `cmd/`(CLI 入口), `server/`(裁判模式服务端), `client/`(玩家模式客户端), `game/`(游戏逻辑)

## 依赖关系
- 各包保持独立，暂无跨包依赖

## 构建与发布
- 二进制名称: `doubletake`，通过 `go build -o doubletake ./cmd/doubletake/` 构建
- Makefile 提供常用命令：`make build`、`make test`、`make cover`、`make lint`、`make clean`、`make snapshot`、`make release`
- GoReleaser 配置: `.goreleaser.yml`，构建 darwin/linux/windows 的 amd64/arm64 二进制
- GitHub Actions: `ci.yml`（push/PR 测试）、`release.yml`（tag 触发 goreleaser 发布）
- 发布流程: `git tag v1.0.0 && git push origin v1.0.0`

## 注意事项
- 所有用户可见的输出信息使用中文
- `.gitignore` 中 `doubletake` 模式会匹配 `cmd/doubletake/` 目录，添加该目录下文件需使用 `git add -f`
