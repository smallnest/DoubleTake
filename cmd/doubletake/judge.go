package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/smallnest/doubletake/client"
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

// RunJudge runs the referee interactive configuration flow.
// out is used for display output; in provides the interactive input source.
func RunJudge(out io.Writer, in io.Reader, port string, stealth bool) GameConfig {
	disp := client.NewDisplay(out, stealth)
	disp.PrintStartup()

	scanner := bufio.NewScanner(in)
	return collectConfig(out, disp, scanner)
}
