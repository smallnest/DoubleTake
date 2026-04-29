package client

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Display provides disguised output functions that make all I/O look like
// system diagnostic tool output rather than a game.
type Display struct {
	out    io.Writer
	stealth bool
}

// NewDisplay creates a Display that writes to w.
// If stealth is true, output is further simplified and [DATA] tags are hidden.
func NewDisplay(w io.Writer, stealth bool) *Display {
	if w == nil {
		w = os.Stdout
	}
	return &Display{out: w, stealth: stealth}
}

// PrintStartup displays a startup banner resembling a diagnostics tool.
func (d *Display) PrintStartup() {
	fmt.Fprintln(d.out, "[sysmon] v2.4.1 — diagnostics mode")
}

// Prompt displays a technical-style prompt like [node-03]$ and reads a line of input.
// It reads from os.Stdin.
func (d *Display) Prompt(nodeID string) string {
	fmt.Fprintf(d.out, "[node-%s]$ ", nodeID)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return ""
	}
	return strings.TrimSpace(scanner.Text())
}

// Info outputs a log line in [INFO] [session:XXXX] msg format.
func (d *Display) Info(session, msg string) {
	if d.stealth {
		fmt.Fprintf(d.out, "[session:%s] %s\n", session, msg)
		return
	}
	fmt.Fprintf(d.out, "[INFO] [session:%s] %s\n", session, msg)
}

// Warn outputs a log line in [WARN] msg format.
func (d *Display) Warn(msg string) {
	if d.stealth {
		fmt.Fprintf(d.out, "warning: %s\n", msg)
		return
	}
	fmt.Fprintf(d.out, "[WARN] %s\n", msg)
}

// Data outputs a log line in [DATA] [node-XX] msg format.
func (d *Display) Data(nodeID, msg string) {
	if d.stealth {
		fmt.Fprintf(d.out, "[node-%s] %s\n", nodeID, msg)
		return
	}
	fmt.Fprintf(d.out, "[DATA] [node-%s] %s\n", nodeID, msg)
}

// PlayerResult holds a single player's game result for display.
type PlayerResult struct {
	Name  string
	Role  string // Chinese display label, e.g. "平民"
	Alive bool
}

// ShowGameResult displays the final game result in disguised format.
func (d *Display) ShowGameResult(winnerLabel string, results []PlayerResult, civilianWord, undercoverWord string) {
	d.Data("00", fmt.Sprintf("游戏结束 — %s 胜利", winnerLabel))
	for _, r := range results {
		status := "存活"
		if !r.Alive {
			status = "已淘汰"
		}
		d.Data("00", fmt.Sprintf("  %s [%s] %s", r.Name, r.Role, status))
	}
	d.Data("00", fmt.Sprintf("平民词语: %s", civilianWord))
	d.Data("00", fmt.Sprintf("卧底词语: %s", undercoverWord))
}

// ListPlayers lists connected players in a grep-like output format.
func (d *Display) ListPlayers(players []string) {
	for i, p := range players {
		fmt.Fprintf(d.out, "connections.list[%d]: %s  status=active\n", i, p)
	}
}
