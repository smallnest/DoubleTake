package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/smallnest/doubletake/client"
	"github.com/smallnest/doubletake/game"
	"github.com/smallnest/doubletake/server"
)

// GameConfig holds the validated game parameters entered by the referee.
type GameConfig struct {
	TotalPlayers int // 总人数 (4-10)
	Undercovers  int // 卧底人数 (1-3)
	Blanks       int // 白板人数 (0+)
}

// readInt reads a line from the scanner, trims whitespace, and parses it as an int.
func readInt(scanner *bufio.Scanner) (int, error) {
	if !scanner.Scan() {
		return 0, fmt.Errorf("unexpected end of input")
	}
	line := strings.TrimSpace(scanner.Text())
	return strconv.Atoi(line)
}

// collectConfig interactively prompts the referee for game parameters and validates them.
// It loops on invalid input, printing specific error reasons via the Display.
func collectConfig(out io.Writer, disp *client.Display, scanner *bufio.Scanner) GameConfig {
	for {
		disp.Info("0000", "输入游戏参数")

		fmt.Fprint(out, "  玩家人数 (4-10): ")
		total := readIntInput(scanner, disp)

		fmt.Fprint(out, "  卧底人数 (1-3): ")
		undercovers := readIntInput(scanner, disp)

		fmt.Fprint(out, "  白板人数 (0+): ")
		blanks := readIntInput(scanner, disp)

		if err := validateConfig(total, undercovers, blanks); err != nil {
			disp.Warn(err.Error())
			continue
		}

		return GameConfig{
			TotalPlayers: total,
			Undercovers:  undercovers,
			Blanks:       blanks,
		}
	}
}

// readIntInput reads an integer from the scanner.
// On parse failure it prints a warning and returns -1 so validation will fail.
func readIntInput(scanner *bufio.Scanner, disp *client.Display) int {
	n, err := readInt(scanner)
	if err != nil {
		disp.Warn("请输入有效的数字")
		return -1
	}
	return n
}

// validateConfig checks game parameters and returns a descriptive error if invalid.
func validateConfig(total, undercovers, blanks int) error {
	if total < 4 || total > 10 {
		return fmt.Errorf("玩家人数 %d 不在合法范围 (4-10)，请重新输入", total)
	}
	if undercovers < 1 || undercovers > 3 {
		return fmt.Errorf("卧底人数 %d 不在合法范围 (1-3)，请重新输入", undercovers)
	}
	if blanks < 0 {
		return fmt.Errorf("白板人数 %d 不能为负数，请重新输入", blanks)
	}
	if undercovers+blanks >= (total+1)/2 {
		return fmt.Errorf("卧底(%d)+白板(%d)=%d，必须少于总人数的一半(%d)，请重新输入", undercovers, blanks, undercovers+blanks, (total+1)/2)
	}
	return nil
}

// RunJudge runs the referee interactive configuration flow and the waiting phase.
// out is used for display output; in provides the interactive input source.
// It returns the GameConfig that was selected.
func RunJudge(out io.Writer, in io.Reader, port string, stealth bool) GameConfig {
	disp := client.NewDisplay(out, stealth)
	disp.PrintStartup()

	scanner := bufio.NewScanner(in)
	cfg := collectConfig(out, disp, scanner)

	srv := server.NewServer(port, cfg.TotalPlayers)
	go srv.Start()
	defer srv.Stop()

	disp.Info("0000", fmt.Sprintf("房间已创建，等待 %d 名玩家加入...", cfg.TotalPlayers))

	// waitingPhase blocks until the referee confirms game start.
	waitingPhase(out, in, disp, scanner, srv, cfg)

	return cfg
}

// waitingPhase displays player join events and reads stdin for "start" commands.
// It blocks until the referee confirms game start or stdin is closed.
func waitingPhase(out io.Writer, in io.Reader, disp *client.Display, scanner *bufio.Scanner, srv *server.Server, cfg GameConfig) {
	// stdinCh delivers lines read from stdin.
	// stdinDone is closed when the stdin goroutine finishes (input exhausted).
	stdinCh := make(chan string, 1)
	stdinDone := make(chan struct{})
	go func() {
		defer close(stdinDone)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			stdinCh <- line
		}
	}()

	for {
		select {
		case evt := <-srv.OnPlayerJoin:
			disp.Info("0000", fmt.Sprintf("%s joined [%d/%d]", evt.Name, evt.Current, evt.Capacity))
			if evt.Current >= cfg.TotalPlayers {
				disp.Info("0000", "人已齐，输入 start 开始游戏")
			}

		case line := <-stdinCh:
			if strings.EqualFold(line, "start") {
				count := srv.PlayerCount()
				if count < 4 {
					disp.Warn(fmt.Sprintf("至少需要 4 人，当前 %d 人", count))
					continue
				}
				if count < cfg.TotalPlayers {
					fmt.Fprintf(out, "  当前 %d/%d 人，确认开始？(Y/N): ", count, cfg.TotalPlayers)
					// Wait for confirmation
					select {
					case confirm := <-stdinCh:
						if strings.EqualFold(confirm, "Y") || confirm == "y" {
							broadcastReady(disp, srv)
							return
						}
						disp.Info("0000", "已取消，继续等待玩家...")
					case <-srv.OnPlayerJoin:
						// A player joined while waiting for confirmation; go back to main loop.
						continue
					case <-stdinDone:
						return
					}
					continue
				}
				// count == cfg.TotalPlayers (or more, shouldn't happen)
				broadcastReady(disp, srv)
				return
			}

		case <-stdinDone:
			return
		}
	}
}

// broadcastReady sends the READY message to all named players.
func broadcastReady(disp *client.Display, srv *server.Server) {
	disp.Info("0000", "游戏开始！")
	srv.BroadcastToNamedPlayers(game.Message{Type: game.MsgReady, Payload: ""})
}
