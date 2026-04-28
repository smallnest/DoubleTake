# cmd/doubletake

## 架构约定
- CLI 入口点，二进制名称 `doubletake`
- 核心逻辑在 `run(stdout, stderr io.Writer, args []string) int` 中，`main()` 只负责调用
- `run` 接受 `io.Writer` 参数，便于测试捕获输出
- 手动解析 flag（不使用 `flag` 包），避免 `flag.Parse()` 调用 `os.Exit` 导致测试无法捕获

## 测试约定
- 测试文件 `main_test.go` 使用 `bytes.Buffer` 捕获 stdout/stderr
- 每个测试用例验证退出码和输出内容
- 覆盖场景：正常参数、无效 role、默认 port、自定义 port、缺失值、未知选项、help
