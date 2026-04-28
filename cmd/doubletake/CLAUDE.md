# cmd/doubletake

## 架构约定
- CLI 入口点，二进制名称 `doubletake`
- 核心逻辑在 `run(stdout, stderr io.Writer, stdin io.Reader, args []string) int` 中，`main()` 只负责调用
- `run` 接受 `io.Writer` 和 `io.Reader` 参数，便于测试捕获输出和模拟输入
- 手动解析 flag（不使用 `flag` 包），避免 `flag.Parse()` 调用 `os.Exit` 导致测试无法捕获
- 通过 `client.NewDisplay(stdout, stealth)` 创建 Display 实例进行输出，不直接 `fmt.Fprintf`
- 角色分发使用 `switch role` 结构，`RunJudge` 和 `RunPlayer` 各自独立处理

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
