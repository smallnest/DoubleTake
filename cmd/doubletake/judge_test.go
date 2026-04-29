package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"io"
	"testing"
	"time"

	"github.com/smallnest/doubletake/client"
	"github.com/smallnest/doubletake/game"
	"github.com/smallnest/doubletake/server"
)

func TestValidateConfig_Valid(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		undercovers int
		blanks      int
	}{
		{"minimum 4 players 1 undercover", 4, 1, 0},
		{"5 players 2 undercovers", 5, 2, 0},
		{"6 players 1 undercover 1 blank", 6, 1, 1},
		{"8 players 2 undercovers 1 blank", 8, 2, 1},
		{"10 players 3 undercovers", 10, 3, 0},
		{"10 players 2 undercovers 2 blanks", 10, 2, 2},
		{"7 players 1 undercover 2 blanks", 7, 1, 2},
		{"9 players 2 undercovers 1 blank", 9, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.total, tt.undercovers, tt.blanks)
			if err != nil {
				t.Errorf("validateConfig(%d, %d, %d) returned error: %v", tt.total, tt.undercovers, tt.blanks, err)
			}
		})
	}
}

func TestValidateConfig_TotalOutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		total int
	}{
		{"too few 3", 3},
		{"too few 1", 1},
		{"too few 0", 0},
		{"too many 11", 11},
		{"too many 15", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.total, 1, 0)
			if err == nil {
				t.Errorf("expected error for total=%d", tt.total)
			}
			if !strings.Contains(err.Error(), "玩家人数") {
				t.Errorf("error should mention 玩家人数, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_UndercoverOutOfRange(t *testing.T) {
	tests := []struct {
		name        string
		undercovers int
	}{
		{"zero undercovers", 0},
		{"negative undercovers", -1},
		{"too many undercovers 4", 4},
		{"too many undercovers 5", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(6, tt.undercovers, 0)
			if err == nil {
				t.Errorf("expected error for undercovers=%d", tt.undercovers)
			}
			if !strings.Contains(err.Error(), "卧底人数") {
				t.Errorf("error should mention 卧底人数, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_NegativeBlanks(t *testing.T) {
	err := validateConfig(6, 1, -1)
	if err == nil {
		t.Error("expected error for negative blanks")
	}
	if !strings.Contains(err.Error(), "白板人数") {
		t.Errorf("error should mention 白板人数, got: %v", err)
	}
}

func TestValidateConfig_TooManySpecialRoles(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		undercovers int
		blanks      int
	}{
		{"4 players 2 undercovers", 4, 2, 0},
		{"5 players 2 undercovers 1 blank", 5, 2, 1},
		{"6 players 3 undercovers", 6, 3, 0},
		{"8 players 3 undercovers 2 blanks", 8, 3, 2},
		{"10 players 3 undercovers 3 blanks", 10, 3, 3},
		{"4 players 1 undercover 1 blank", 4, 1, 1},
		{"6 players 2 undercovers 1 blank", 6, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.total, tt.undercovers, tt.blanks)
			if err == nil {
				t.Errorf("expected error for total=%d undercovers=%d blanks=%d", tt.total, tt.undercovers, tt.blanks)
			}
			if !strings.Contains(err.Error(), "卧底") || !strings.Contains(err.Error(), "白板") {
				t.Errorf("error should mention 卧底 and 白板, got: %v", err)
			}
		})
	}
}

func TestCollectConfig_ValidInput(t *testing.T) {
	input := "6\n1\n1\n"
	out := &bytes.Buffer{}
	cfg := runCollectConfig(out, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
	if cfg.Undercovers != 1 {
		t.Errorf("expected undercovers 1, got %d", cfg.Undercovers)
	}
	if cfg.Blanks != 1 {
		t.Errorf("expected blanks 1, got %d", cfg.Blanks)
	}
}

func TestCollectConfig_InvalidTotalThenValid(t *testing.T) {
	input := "3\n1\n0\n6\n1\n0\n"
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
	if cfg.Undercovers != 1 {
		t.Errorf("expected undercovers 1, got %d", cfg.Undercovers)
	}
	if cfg.Blanks != 0 {
		t.Errorf("expected blanks 0, got %d", cfg.Blanks)
	}
}

func TestCollectConfig_InvalidUndercoverThenValid(t *testing.T) {
	input := "6\n4\n0\n6\n1\n0\n"
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_NegativeBlanksThenValid(t *testing.T) {
	input := "6\n1\n-1\n6\n1\n0\n"
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.Blanks != 0 {
		t.Errorf("expected blanks 0, got %d", cfg.Blanks)
	}
}

func TestCollectConfig_TooManySpecialRolesThenValid(t *testing.T) {
	input := "4\n1\n1\n6\n1\n0\n"
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_MultipleRetries(t *testing.T) {
	input := "3\n1\n0\n" + // total too low
		"6\n0\n0\n" + // undercover too low
		"6\n1\n-1\n" + // blanks negative
		"6\n1\n0\n" // valid
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
	if cfg.Undercovers != 1 {
		t.Errorf("expected undercovers 1, got %d", cfg.Undercovers)
	}
	if cfg.Blanks != 0 {
		t.Errorf("expected blanks 0, got %d", cfg.Blanks)
	}
}

func TestCollectConfig_BoundaryMaxPlayers(t *testing.T) {
	input := "10\n1\n0\n"
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 10 {
		t.Errorf("expected total 10, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_BoundaryMinPlayers(t *testing.T) {
	input := "4\n1\n0\n"
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 4 {
		t.Errorf("expected total 4, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_WarnsOnInvalidTotal(t *testing.T) {
	out := &bytes.Buffer{}
	input := "3\n1\n0\n6\n1\n0\n"
	runCollectConfig(out, input)

	output := out.String()
	if !strings.Contains(output, "玩家人数 3 不在合法范围") {
		t.Errorf("expected warning about total players, got: %s", output)
	}
}

func TestCollectConfig_WarnsOnInvalidUndercover(t *testing.T) {
	out := &bytes.Buffer{}
	input := "6\n0\n0\n6\n1\n0\n"
	runCollectConfig(out, input)

	output := out.String()
	if !strings.Contains(output, "卧底人数 0 不在合法范围") {
		t.Errorf("expected warning about undercover count, got: %s", output)
	}
}

func TestCollectConfig_WarnsOnNegativeBlanks(t *testing.T) {
	out := &bytes.Buffer{}
	input := "6\n1\n-1\n6\n1\n0\n"
	runCollectConfig(out, input)

	output := out.String()
	if !strings.Contains(output, "白板人数 -1 不能为负数") {
		t.Errorf("expected warning about negative blanks, got: %s", output)
	}
}

func TestCollectConfig_WarnsOnTooManySpecialRoles(t *testing.T) {
	out := &bytes.Buffer{}
	input := "4\n1\n1\n6\n1\n0\n"
	runCollectConfig(out, input)

	output := out.String()
	if !strings.Contains(output, "卧底(1)+白板(1)=2，必须少于总人数的一半(2)") {
		t.Errorf("expected warning about special roles ratio, got: %s", output)
	}
}

func TestReadInt_ValidInput(t *testing.T) {
	scanner := newTestScanner("42\n")
	n, err := readInt(scanner)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
}

func TestReadInt_TrimsWhitespace(t *testing.T) {
	scanner := newTestScanner("  7  \n")
	n, err := readInt(scanner)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 7 {
		t.Errorf("expected 7, got %d", n)
	}
}

func TestReadInt_InvalidInput(t *testing.T) {
	scanner := newTestScanner("abc\n")
	_, err := readInt(scanner)
	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}

func TestReadInt_EmptyInput(t *testing.T) {
	scanner := newTestScanner("")
	_, err := readInt(scanner)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestCollectConfig_NonNumericInput(t *testing.T) {
	// First iteration: total=abc → -1, undercover=4, blanks=4 -> validation fails (total<4)
	// Second iteration: total=6, undercover=1, blanks=0 -> valid
	input := "abc\n4\n4\n6\n1\n0\n"
	out := &bytes.Buffer{}
	cfg := runCollectConfig(out, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_ZeroBlanksAllowed(t *testing.T) {
	input := "4\n1\n0\n"
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.Blanks != 0 {
		t.Errorf("expected blanks 0, got %d", cfg.Blanks)
	}
	if cfg.TotalPlayers != 4 {
		t.Errorf("expected total 4, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_LargeValidGame(t *testing.T) {
	input := "10\n3\n1\n"
	cfg := runCollectConfig(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 10 {
		t.Errorf("expected total 10, got %d", cfg.TotalPlayers)
	}
	if cfg.Undercovers != 3 {
		t.Errorf("expected undercovers 3, got %d", cfg.Undercovers)
	}
	if cfg.Blanks != 1 {
		t.Errorf("expected blanks 1, got %d", cfg.Blanks)
	}
}

// --- Waiting phase integration tests ---

func TestWaitingPhase_PlayerJoinNotification(t *testing.T) {
	out, port, cleanup := startJudgeForTest(t, "4\n1\n0\n")
	defer cleanup()

	// Connect a player
	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "JOIN|Alice\n")

	// Wait for the join notification to appear in output
	waitForOutput(t, out, "Alice joined [1/4]", 2*time.Second)
}

func TestWaitingPhase_AllPlayersJoined(t *testing.T) {
	out, port, cleanup := startJudgeForTest(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"Alice", "Bob", "Charlie", "Dave"}
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	// Should see "人已齐" message
	waitForOutput(t, out, "人已齐，输入 start 开始游戏", 3*time.Second)
}

func TestWaitingPhase_StartWithFewerThan4(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	// Connect only 3 players
	for _, name := range []string{"A", "B", "C"} {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "C joined [3/4]", 2*time.Second)

	// Try to start with fewer than 4 players
	stdin <- "start"

	waitForOutput(t, out, "至少需要 4 人，当前 3 人", 2*time.Second)
}

func TestWaitingPhase_StartWithConfirmation(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "6\n1\n0\n")
	defer cleanup()

	// Connect 4 players (less than 6)
	for _, name := range []string{"A", "B", "C", "D"} {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "D joined [4/6]", 2*time.Second)

	// Start with confirmation
	stdin <- "start"
	waitForOutput(t, out, "当前 4/6 人，确认开始？(Y/N)", 2*time.Second)
	stdin <- "N"
	waitForOutput(t, out, "已取消，继续等待玩家...", 2*time.Second)
}

func TestWaitingPhase_StartConfirmed(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"A", "B", "C", "D"}
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "人已齐，输入 start 开始游戏", 2*time.Second)

	// Consume JOIN confirmation from each connection
	for _, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(time.Second))
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("failed to consume JOIN confirmation: %v", err)
		}
		msg, err := game.Decode(line)
		if err != nil {
			t.Fatalf("failed to decode JOIN confirmation: %v", err)
		}
		if msg.Type != game.MsgJoin {
			t.Fatalf("expected JOIN confirmation, got %s", msg.Type)
		}
	}

	// Start with full capacity — no confirmation needed
	stdin <- "start"
	stdin <- "苹果"
	stdin <- "香蕉"

	// Read ROLE and READY from each connection, verify role/word content
	type roleInfo struct {
		roleName string
		word     string
	}
	var roles []roleInfo
	for _, conn := range conns {
		reader := bufio.NewReader(conn)

		conn.SetReadDeadline(time.Now().Add(time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("did not receive ROLE: %v", err)
			continue
		}
		msg, err := game.Decode(line)
		if err != nil {
			t.Errorf("decode error for ROLE: %v", err)
			continue
		}
		if msg.Type != game.MsgRole {
			t.Errorf("expected ROLE, got %s", msg.Type)
		}

		parts := strings.SplitN(msg.Payload, "|", 2)
		if len(parts) < 2 {
			t.Errorf("ROLE payload malformed: %q", msg.Payload)
			continue
		}
		roleName, word := parts[0], parts[1]
		roles = append(roles, roleInfo{roleName, word})

		switch roleName {
		case "Civilian":
			if word != "苹果" {
				t.Errorf("Civilian got word %q, want %q", word, "苹果")
			}
		case "Undercover":
			if word != "香蕉" {
				t.Errorf("Undercover got word %q, want %q", word, "香蕉")
			}
		default:
			t.Errorf("unexpected role %q", roleName)
		}

		conn.SetReadDeadline(time.Now().Add(time.Second))
		line, err = reader.ReadString('\n')
		if err != nil {
			t.Errorf("did not receive READY: %v", err)
			continue
		}
		msg, err = game.Decode(line)
		if err != nil {
			t.Errorf("decode error for READY: %v", err)
			continue
		}
		if msg.Type != game.MsgReady {
			t.Errorf("expected READY, got %s", msg.Type)
		}
	}

	// Verify role distribution: 4 players, 1U, 0B → 3 civilians, 1 undercover
	civilianCount := 0
	undercoverCount := 0
	for _, r := range roles {
		switch r.roleName {
		case "Civilian":
			civilianCount++
		case "Undercover":
			undercoverCount++
		}
	}
	if civilianCount != 3 {
		t.Errorf("expected 3 civilians, got %d", civilianCount)
	}
	if undercoverCount != 1 {
		t.Errorf("expected 1 undercover, got %d", undercoverCount)
	}

	waitForOutput(t, out, "游戏开始！", 2*time.Second)
}

func TestWaitingPhase_WithBlankPlayer(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "5\n1\n1\n")
	defer cleanup()

	names := []string{"A", "B", "C", "D", "E"}
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "人已齐，输入 start 开始游戏", 2*time.Second)

	// Consume JOIN confirmation from each connection
	for _, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(time.Second))
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("failed to consume JOIN: %v", err)
		}
		msg, err := game.Decode(line)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if msg.Type != game.MsgJoin {
			t.Fatalf("expected JOIN, got %s", msg.Type)
		}
	}

	stdin <- "start"
	stdin <- "苹果"
	stdin <- "香蕉"

	// Read ROLE from each connection, verify role/word content including blank
	type roleInfo struct {
		roleName string
		word     string
	}
	var roles []roleInfo
	for _, conn := range conns {
		reader := bufio.NewReader(conn)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("did not receive ROLE: %v", err)
			continue
		}
		msg, err := game.Decode(line)
		if err != nil {
			t.Errorf("decode error: %v", err)
			continue
		}
		if msg.Type != game.MsgRole {
			t.Errorf("expected ROLE, got %s", msg.Type)
			continue
		}

		parts := strings.SplitN(msg.Payload, "|", 2)
		if len(parts) < 2 {
			t.Errorf("malformed ROLE payload: %q", msg.Payload)
			continue
		}
		roles = append(roles, roleInfo{parts[0], parts[1]})
	}

	// Count and verify each role's word
	counts := map[string]int{"Civilian": 0, "Undercover": 0, "Blank": 0}
	for _, r := range roles {
		counts[r.roleName]++
		switch r.roleName {
		case "Civilian":
			if r.word != "苹果" {
				t.Errorf("Civilian got word %q, want %q", r.word, "苹果")
			}
		case "Undercover":
			if r.word != "香蕉" {
				t.Errorf("Undercover got word %q, want %q", r.word, "香蕉")
			}
		case "Blank":
			if r.word != "你是白板" {
				t.Errorf("Blank got word %q, want %q", r.word, "你是白板")
			}
		default:
			t.Errorf("unexpected role %q", r.roleName)
		}
	}
	if counts["Civilian"] != 3 {
		t.Errorf("expected 3 civilians, got %d", counts["Civilian"])
	}
	if counts["Undercover"] != 1 {
		t.Errorf("expected 1 undercover, got %d", counts["Undercover"])
	}
	if counts["Blank"] != 1 {
		t.Errorf("expected 1 blank, got %d", counts["Blank"])
	}

	waitForOutput(t, out, "游戏开始！", 2*time.Second)
}

func TestWaitingPhase_SameWordRetry(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"A", "B", "C", "D"}
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "人已齐，输入 start 开始游戏", 2*time.Second)

	// Consume JOIN confirmation from each connection
	for _, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(time.Second))
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("failed to consume JOIN: %v", err)
		}
		if _, err := game.Decode(line); err != nil {
			t.Fatalf("failed to decode JOIN: %v", err)
		}
	}

	// Send start and words — first pair same to trigger retry
	stdin <- "start"
	stdin <- "苹果" // first civilian attempt
	stdin <- "苹果" // first undercover attempt (same!)

	waitForOutput(t, out, "不能相同", 2*time.Second)

	// Send different words
	stdin <- "苹果" // second civilian attempt
	stdin <- "香蕉" // second undercover attempt (different)

	// Read ROLE and READY from each connection
	type roleInfo struct {
		roleName string
		word     string
	}
	var roles []roleInfo
	for _, conn := range conns {
		reader := bufio.NewReader(conn)

		conn.SetReadDeadline(time.Now().Add(time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("did not receive ROLE: %v", err)
			continue
		}
		msg, err := game.Decode(line)
		if err != nil {
			t.Errorf("decode error for ROLE: %v", err)
			continue
		}
		if msg.Type != game.MsgRole {
			t.Errorf("expected ROLE, got %s", msg.Type)
			continue
		}

		parts := strings.SplitN(msg.Payload, "|", 2)
		if len(parts) < 2 {
			t.Errorf("malformed ROLE payload: %q", msg.Payload)
			continue
		}
		roles = append(roles, roleInfo{parts[0], parts[1]})

		switch parts[0] {
		case "Civilian":
			if parts[1] != "苹果" {
				t.Errorf("Civilian got word %q, want %q", parts[1], "苹果")
			}
		case "Undercover":
			if parts[1] != "香蕉" {
				t.Errorf("Undercover got word %q, want %q", parts[1], "香蕉")
			}
		default:
			t.Errorf("unexpected role %q", parts[0])
		}

		conn.SetReadDeadline(time.Now().Add(time.Second))
		line, err = reader.ReadString('\n')
		if err != nil {
			t.Errorf("did not receive READY: %v", err)
			continue
		}
		msg, err = game.Decode(line)
		if err != nil {
			t.Errorf("decode error for READY: %v", err)
			continue
		}
		if msg.Type != game.MsgReady {
			t.Errorf("expected READY, got %s", msg.Type)
		}
	}

	// Verify role distribution
	civilianCount := 0
	undercoverCount := 0
	for _, r := range roles {
		switch r.roleName {
		case "Civilian":
			civilianCount++
		case "Undercover":
			undercoverCount++
		}
	}
	if civilianCount != 3 {
		t.Errorf("expected 3 civilians, got %d", civilianCount)
	}
	if undercoverCount != 1 {
		t.Errorf("expected 1 undercover, got %d", undercoverCount)
	}

	waitForOutput(t, out, "游戏开始！", 2*time.Second)
}

func TestWaitingPhase_EndToEnd(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "6\n1\n1\n")
	defer cleanup()

	names := []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank"}
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "人已齐，输入 start 开始游戏", 2*time.Second)

	// Consume JOIN confirmation from each connection
	for _, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(time.Second))
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("failed to consume JOIN: %v", err)
		}
		msg, err := game.Decode(line)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if msg.Type != game.MsgJoin {
			t.Fatalf("expected JOIN, got %s", msg.Type)
		}
	}

	stdin <- "start"
	stdin <- "电脑"
	stdin <- "计算器"

	// Read ROLE and READY per connection (sequentially to avoid server closing before all reads)
	type roleInfo struct {
		roleName string
		word     string
	}
	var roles []roleInfo
	for _, conn := range conns {
		reader := bufio.NewReader(conn)

		conn.SetReadDeadline(time.Now().Add(time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("did not receive ROLE: %v", err)
			continue
		}
		msg, err := game.Decode(line)
		if err != nil {
			t.Errorf("decode error: %v", err)
			continue
		}
		if msg.Type != game.MsgRole {
			t.Errorf("expected ROLE, got %s", msg.Type)
			continue
		}

		parts := strings.SplitN(msg.Payload, "|", 2)
		if len(parts) < 2 {
			t.Errorf("malformed ROLE payload: %q", msg.Payload)
			continue
		}
		roles = append(roles, roleInfo{parts[0], parts[1]})

		switch parts[0] {
		case "Civilian":
			if parts[1] != "电脑" {
				t.Errorf("Civilian got word %q, want %q", parts[1], "电脑")
			}
		case "Undercover":
			if parts[1] != "计算器" {
				t.Errorf("Undercover got word %q, want %q", parts[1], "计算器")
			}
		case "Blank":
			if parts[1] != "你是白板" {
				t.Errorf("Blank got word %q, want %q", parts[1], "你是白板")
			}
		default:
			t.Errorf("unexpected role %q", parts[0])
		}

		// Read READY before moving to next connection
		conn.SetReadDeadline(time.Now().Add(time.Second))
		line, err = reader.ReadString('\n')
		if err != nil {
			t.Errorf("did not receive READY: %v", err)
			continue
		}
		msg, err = game.Decode(line)
		if err != nil {
			t.Errorf("decode error for READY: %v", err)
			continue
		}
		if msg.Type != game.MsgReady {
			t.Errorf("expected READY, got %s", msg.Type)
		}
	}

	// Verify role distribution: 6 players, 1U, 1B → 4 civilians, 1 undercover, 1 blank
	counts := map[string]int{"Civilian": 0, "Undercover": 0, "Blank": 0}
	for _, r := range roles {
		counts[r.roleName]++
	}
	if counts["Civilian"] != 4 {
		t.Errorf("expected 4 civilians, got %d", counts["Civilian"])
	}
	if counts["Undercover"] != 1 {
		t.Errorf("expected 1 undercover, got %d", counts["Undercover"])
	}
	if counts["Blank"] != 1 {
		t.Errorf("expected 1 blank, got %d", counts["Blank"])
	}

	waitForOutput(t, out, "游戏开始！", 2*time.Second)
}

func TestWaitingPhase_StartWithPartialConfirmation(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "6\n1\n0\n")
	defer cleanup()

	// Connect 5 players (>=4 but <6)
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "E joined [5/6]", 2*time.Second)

	// Start and confirm with Y
	stdin <- "start"
	waitForOutput(t, out, "当前 5/6 人，确认开始？(Y/N)", 2*time.Second)
	stdin <- "Y"
	// Provide words for collectWords phase
	stdin <- "苹果"
	stdin <- "香蕉"
	waitForOutput(t, out, "游戏开始！", 2*time.Second)
}

// --- collectWords tests ---

func TestCollectWords_Valid(t *testing.T) {
	out := &bytes.Buffer{}
	disp := newDisplay(out)
	scanner := newTestScanner("苹果\n香蕉\n")

	civilian, undercover := collectWords(out, disp, scanner)
	if civilian != "苹果" {
		t.Errorf("civilian word = %q, want %q", civilian, "苹果")
	}
	if undercover != "香蕉" {
		t.Errorf("undercover word = %q, want %q", undercover, "香蕉")
	}
}

func TestCollectWords_SameWordsRetry(t *testing.T) {
	out := &bytes.Buffer{}
	disp := newDisplay(out)
	// First attempt: same words, second attempt: different words
	scanner := newTestScanner("苹果\n苹果\n苹果\n香蕉\n")

	civilian, undercover := collectWords(out, disp, scanner)
	if civilian != "苹果" {
		t.Errorf("civilian word = %q, want %q", civilian, "苹果")
	}
	if undercover != "香蕉" {
		t.Errorf("undercover word = %q, want %q", undercover, "香蕉")
	}

	output := out.String()
	if !strings.Contains(output, "不能相同") {
		t.Errorf("expected warning about same words, got: %s", output)
	}
}

func TestCollectWords_EmptyWordRetry(t *testing.T) {
	out := &bytes.Buffer{}
	disp := newDisplay(out)
	// First attempt: empty civilian word, second: valid
	scanner := newTestScanner("\n香蕉\n苹果\n香蕉\n")

	civilian, undercover := collectWords(out, disp, scanner)
	if civilian != "苹果" {
		t.Errorf("civilian word = %q, want %q", civilian, "苹果")
	}
	if undercover != "香蕉" {
		t.Errorf("undercover word = %q, want %q", undercover, "香蕉")
	}

	output := out.String()
	if !strings.Contains(output, "不能为空") {
		t.Errorf("expected warning about empty words, got: %s", output)
	}
}

func TestCollectWords_EOF(t *testing.T) {
	out := &bytes.Buffer{}
	disp := newDisplay(out)
	scanner := newTestScanner("")

	civilian, undercover := collectWords(out, disp, scanner)
	if civilian != "" || undercover != "" {
		t.Errorf("expected empty words on EOF, got %q/%q", civilian, undercover)
	}
}

// --- Test helpers ---

// newTestScanner creates a bufio.Scanner from a string for testing.
func newTestScanner(input string) *bufio.Scanner {
	return bufio.NewScanner(strings.NewReader(input))
}

// runCollectConfig is a test helper that runs collectConfig with the given input.
func runCollectConfig(out *bytes.Buffer, input string) GameConfig {
	disp := newDisplay(out)
	scanner := bufio.NewScanner(strings.NewReader(input))
	return collectConfig(out, disp, scanner)
}

// newDisplay creates a Display for testing.
func newDisplay(out *bytes.Buffer) *client.Display {
	return client.NewDisplay(out, false)
}

// startJudgeForTest runs RunJudge in a goroutine and returns a buffered output, the server port, and a cleanup function.
func startJudgeForTest(t *testing.T, configInput string) (*safeBuffer, string, func()) {
	t.Helper()
	return startJudgeForTestWithStdinRaw(t, configInput)
}

// startJudgeForTestWithStdin runs RunJudge and also returns a channel to inject stdin lines.
func startJudgeForTestWithStdin(t *testing.T, configInput string) (*safeBuffer, string, chan string, func()) {
	t.Helper()
	stdinCh := make(chan string, 16)
	out, port, cleanup := startJudgeForTestWithStdinRaw(t, configInput, stdinCh)
	return out, port, stdinCh, cleanup
}

func startJudgeForTestWithStdinRaw(t *testing.T, configInput string, extraLines ...chan string) (*safeBuffer, string, func()) {
	t.Helper()

	// Find a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	out := &safeBuffer{}

	// Build a reader that first provides config, then lines from extraLines channel
	reader, writer := net.Pipe()

	// Write config input
	go func() {
		writer.Write([]byte(configInput))
		// If there's an extraLines channel, feed from it
		if len(extraLines) > 0 {
			for line := range extraLines[0] {
				writer.Write([]byte(line + "\n"))
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		RunJudge(out, reader, port, false)
		close(done)
	}()

	// Wait for server to be listening
	for i := 0; i < 100; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 10*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cleanup := func() {
		// Close extraLines channel so the stdin goroutine in RunJudge can detect EOF
		for _, ch := range extraLines {
			close(ch)
		}
		writer.Close()
		reader.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}

	return out, port, cleanup
}

// safeBuffer is a thread-safe bytes.Buffer for concurrent read/write in tests.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

// waitForOutput polls the safeBuffer until the expected substring appears or times out.
func waitForOutput(t *testing.T, out *safeBuffer, substr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), substr) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in output: %s", substr, out.String())
}

// --- Description phase integration tests ---

// findConnByName finds the connection for a player by name (P0, P1, ...).
func findConnByName(conns []net.Conn, name string) net.Conn {
	for i, conn := range conns {
		if fmt.Sprintf("P%d", i) == name {
			return conn
		}
	}
	return nil
}

// consumeMsgs reads and discards the specified number of messages from a connection.
func consumeMsgs(t *testing.T, conn net.Conn, count int) {
	t.Helper()
	reader := bufio.NewReader(conn)
	for i := 0; i < count; i++ {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("consumeMsgs: failed to read msg %d/%d: %v", i+1, count, err)
		}
		_, _ = game.Decode(line)
	}
}

// readerForConn returns a bufio.Reader for the given connection, creating one if needed.
// This avoids data loss from creating multiple bufio.Readers on the same conn.
func readerForConn(conn net.Conn) *bufio.Reader {
	type readerKey struct{}
	// Use a simple approach: store in a package-level map.
	readersMu.Lock()
	defer readersMu.Unlock()
	if r, ok := readers[conn]; ok {
		return r
	}
	r := bufio.NewReader(conn)
	readers[conn] = r
	return r
}

var (
	readers   = make(map[net.Conn]*bufio.Reader)
	readersMu sync.Mutex
)

// readMsgFromConn reads one message from a connection with timeout.
func readMsgFromConn(t *testing.T, conn net.Conn) game.Message {
	t.Helper()
	reader := readerForConn(conn)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}
	msg, err := game.Decode(line)
	if err != nil {
		t.Fatalf("failed to decode message: %v", err)
	}
	return msg
}

// setupDescPhaseTest is a test helper that starts a judge, connects players,
// and advances through the config, waiting, word collection, and role assignment
// phases. It returns the player connections and cleanup function.
// After this returns, all connections have consumed JOIN, ROLE, and READY messages,
// and are ready for the description phase.
func setupDescPhaseTest(t *testing.T, numPlayers int) ([]net.Conn, func()) {
	t.Helper()

	// 1U 0B for any count >= 4
	configInput := fmt.Sprintf("%d\n1\n0\n", numPlayers)
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, configInput)

	names := make([]string, numPlayers)
	for i := 0; i < numPlayers; i++ {
		names[i] = fmt.Sprintf("P%d", i)
	}

	conns := make([]net.Conn, numPlayers)
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "人已齐，输入 start 开始游戏", 2*time.Second)

	// Consume JOIN confirmation from each connection.
	for _, conn := range conns {
		readMsgFromConn(t, conn) // JOIN confirmation
	}

	// Start the game.
	stdin <- "start"
	stdin <- "苹果"
	stdin <- "香蕉"

	// Each connection gets ROLE + READY. Consume them.
	for i, conn := range conns {
		msg1 := readMsgFromConn(t, conn)
		t.Logf("conn %d: msg1=%s|%s", i, msg1.Type, msg1.Payload)
		msg2 := readMsgFromConn(t, conn)
		t.Logf("conn %d: msg2=%s|%s", i, msg2.Type, msg2.Payload)
	}

	waitForOutput(t, out, "游戏开始！", 2*time.Second)

	// Return conns and a cleanup that closes stdin.
	// NOTE: we don't close stdin here because the description phase
	// blocks on srv.OnDescMsg, not stdin. The caller must send DESC messages.
	_ = stdin
	return conns, cleanup
}

func TestDescriptionPhase_NormalFlow(t *testing.T) {
	conns, cleanup := setupDescPhaseTest(t, 4)
	defer cleanup()

	t.Logf("setup complete, reading messages")

	// All players should receive ROUND|1|P0,P1,P2,P3 (order may vary due to shuffle).
	// For this test we'll just verify the message types.
	// The speaker order is random due to AssignRoles shuffle, so we read ROUND to learn it.
	roundMsg := readMsgFromConn(t, conns[0])
	if roundMsg.Type != game.MsgRound {
		t.Fatalf("expected ROUND, got %s", roundMsg.Type)
	}
	// Parse round number and speaker order from ROUND|1|P0,P1,...
	roundParts := strings.SplitN(roundMsg.Payload, "|", 2)
	if len(roundParts) < 2 {
		t.Fatalf("malformed ROUND payload: %q", roundMsg.Payload)
	}
	speakers := strings.Split(roundParts[1], ",")

	// All players should also get the first TURN message.
	turnMsg := readMsgFromConn(t, conns[0])
	if turnMsg.Type != game.MsgTurn {
		t.Fatalf("expected TURN, got %s", turnMsg.Type)
	}

	// Consume ROUND+TURN from remaining connections.
	for i := 1; i < len(conns); i++ {
		msg := readMsgFromConn(t, conns[i])
		if msg.Type != game.MsgRound {
			t.Errorf("conn %d: expected ROUND, got %s", i, msg.Type)
		}
		msg = readMsgFromConn(t, conns[i])
		if msg.Type != game.MsgTurn {
			t.Errorf("conn %d: expected TURN, got %s", i, msg.Type)
		}
	}

	// Each speaker describes in order.
	for si, speaker := range speakers {
		speakerConn := findConnByName(conns, speaker)

		// Send DESC from the current speaker.
		fmt.Fprintf(speakerConn, "DESC|description from %s\n", speaker)

		// All players should receive DESC|speaker|description broadcast.
		for _, conn := range conns {
			msg := readMsgFromConn(t, conn)
			if msg.Type != game.MsgDesc {
				t.Fatalf("expected DESC broadcast, got %s: %s", msg.Type, msg.Payload)
			}
		}

		// After DESC, a TURN for the next speaker is broadcast (unless this is the last speaker).
		if si < len(speakers)-1 {
			for _, conn := range conns {
				msg := readMsgFromConn(t, conn)
				if msg.Type != game.MsgTurn {
					t.Fatalf("expected TURN for next speaker, got %s: %s", msg.Type, msg.Payload)
 				}
 			}
 		}
 	}
 }

// --- PK phase integration tests ---

// assertMsgFromAll reads one message from every connection and asserts the type matches.
func assertMsgFromAll(t *testing.T, conns []net.Conn, expectedType string) game.Message {
	t.Helper()
	var firstMsg game.Message
	for i, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if i == 0 {
			firstMsg = msg
		}
		if msg.Type != expectedType {
			t.Errorf("conn %d: expected %s, got %s: %s", i, expectedType, msg.Type, msg.Payload)
		}
	}
	return firstMsg
}

// setupPKTestEnv creates a server with connected and named players for direct pkPhase testing.
func setupPKTestEnv(t *testing.T, numPlayers int) (*server.Server, []net.Conn, *client.Display, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	srv := server.NewServer(port, numPlayers)
	go srv.Start()

	// Wait for server to be ready
	for i := 0; i < 100; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 10*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Connect and register players
	conns := make([]net.Conn, numPlayers)
	for i := 0; i < numPlayers; i++ {
		name := fmt.Sprintf("P%d", i)
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		fmt.Fprintf(conn, "JOIN|%s\n", name)
		// Consume JOIN confirmation
		readerForConn(conn)
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgJoin {
			t.Fatalf("expected JOIN confirmation, got %s", msg.Type)
		}
	}

	disp := client.NewDisplay(&bytes.Buffer{}, false)

	cleanup := func() {
		for _, conn := range conns {
			conn.Close()
		}
		srv.Stop()
	}

	return srv, conns, disp, cleanup
}

func TestPKPhase_SingleRound(t *testing.T) {
	srv, conns, disp, cleanup := setupPKTestEnv(t, 4)
	defer cleanup()

	players := []*game.Player{
		{Name: "P0", Alive: true},
		{Name: "P1", Alive: true},
		{Name: "P2", Alive: true},
		{Name: "P3", Alive: true},
	}

	tiedPlayers := []string{"P0", "P1"}

	resultCh := make(chan string, 1)
	go func() {
		resultCh <- pkPhase(disp, srv, tiedPlayers, players, make(chan struct{}))
	}()

	// 1. All players receive PK_START|P0,P1
	assertMsgFromAll(t, conns, game.MsgPKStart)

	// --- Description phase for tied players ---
	// 2. ROUND|pkNum|P0,P1
	roundMsg := assertMsgFromAll(t, conns, game.MsgRound)
	if !strings.Contains(roundMsg.Payload, "P0,P1") {
		t.Errorf("ROUND should contain P0,P1, got %s", roundMsg.Payload)
	}

	// 3. TURN|P0 (first tied speaker)
	assertMsgFromAll(t, conns, game.MsgTurn)

	// 4. P0 describes
	fmt.Fprintf(conns[0], "DESC|desc from P0\n")

	// 5. All receive DESC|P0|desc from P0
	descMsg := assertMsgFromAll(t, conns, game.MsgDesc)
	if !strings.Contains(descMsg.Payload, "P0") {
		t.Errorf("DESC should contain P0, got %s", descMsg.Payload)
	}

	// 6. TURN|P1 (next speaker)
	assertMsgFromAll(t, conns, game.MsgTurn)

	// 7. P1 describes
	fmt.Fprintf(conns[1], "DESC|desc from P1\n")

	// 8. All receive DESC|P1|desc from P1
	descMsg2 := assertMsgFromAll(t, conns, game.MsgDesc)
	if !strings.Contains(descMsg2.Payload, "P1") {
		t.Errorf("DESC should contain P1, got %s", descMsg2.Payload)
	}

	// --- PK voting phase ---
	// 9. ROUND|pkNum|voterList (P0,P1,P2,P3)
	assertMsgFromAll(t, conns, game.MsgRound)

	// 10. TURN|P0 (first voter)
	assertMsgFromAll(t, conns, game.MsgTurn)

	// 11. P0 votes for P1
	fmt.Fprintf(conns[0], "PK_VOTE|P1\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast) // VOTE_BC|P0|P1

	// 12. TURN|P1 (second voter)
	assertMsgFromAll(t, conns, game.MsgTurn)

	// 13. P1 votes for P0
	fmt.Fprintf(conns[1], "PK_VOTE|P0\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	// 14. TURN|P2
	assertMsgFromAll(t, conns, game.MsgTurn)

	// 15. P2 votes for P1
	fmt.Fprintf(conns[2], "PK_VOTE|P1\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	// 16. TURN|P3
	assertMsgFromAll(t, conns, game.MsgTurn)

	// 17. P3 votes for P1 → P1 gets 3 votes, P0 gets 1 → P1 eliminated
	fmt.Fprintf(conns[3], "PK_VOTE|P1\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	// 18. RESULT|tally
	resultMsg := assertMsgFromAll(t, conns, game.MsgResult)
	t.Logf("PK RESULT: %s", resultMsg.Payload)

	// 19. KICK|P1
	kickMsg := assertMsgFromAll(t, conns, game.MsgKick)
	if kickMsg.Payload != "P1" {
		t.Errorf("expected KICK P1, got %s", kickMsg.Payload)
	}

	eliminated := <-resultCh
	if eliminated != "P1" {
		t.Errorf("pkPhase returned %q, want %q", eliminated, "P1")
	}

	// Verify P1 is marked as not alive
	for _, p := range players {
		if p.Name == "P1" && p.Alive {
			t.Error("P1 should be marked as not alive")
		}
	}
}

func TestPKPhase_MultiRound(t *testing.T) {
	srv, conns, disp, cleanup := setupPKTestEnv(t, 4)
	defer cleanup()

	players := []*game.Player{
		{Name: "P0", Alive: true},
		{Name: "P1", Alive: true},
		{Name: "P2", Alive: true},
		{Name: "P3", Alive: true},
	}

	tiedPlayers := []string{"P0", "P1"}

	resultCh := make(chan string, 1)
	go func() {
		resultCh <- pkPhase(disp, srv, tiedPlayers, players, make(chan struct{}))
	}()

	// === Round 1: tie ===

	// PK_START
	assertMsgFromAll(t, conns, game.MsgPKStart)

	// Round 1 desc: ROUND
	roundMsg := assertMsgFromAll(t, conns, game.MsgRound)
	if !strings.Contains(roundMsg.Payload, "P0,P1") {
		t.Errorf("ROUND should contain P0,P1, got %s", roundMsg.Payload)
	}

	// TURN|P0
	assertMsgFromAll(t, conns, game.MsgTurn)
	fmt.Fprintf(conns[0], "DESC|P0 desc\n")
	assertMsgFromAll(t, conns, game.MsgDesc)

	// TURN|P1
	assertMsgFromAll(t, conns, game.MsgTurn)
	fmt.Fprintf(conns[1], "DESC|P1 desc\n")
	assertMsgFromAll(t, conns, game.MsgDesc)

	// PK voting ROUND
	assertMsgFromAll(t, conns, game.MsgRound)

	// PK voting: P0→P1, P1→P0, P2→P0, P3→P1 → P0=2, P1=2 → tie
	assertMsgFromAll(t, conns, game.MsgTurn) // TURN|P0
	fmt.Fprintf(conns[0], "PK_VOTE|P1\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	assertMsgFromAll(t, conns, game.MsgTurn) // TURN|P1
	fmt.Fprintf(conns[1], "PK_VOTE|P0\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	assertMsgFromAll(t, conns, game.MsgTurn) // TURN|P2
	fmt.Fprintf(conns[2], "PK_VOTE|P0\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	assertMsgFromAll(t, conns, game.MsgTurn) // TURN|P3
	fmt.Fprintf(conns[3], "PK_VOTE|P1\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	// RESULT (tie)
	resultMsg := assertMsgFromAll(t, conns, game.MsgResult)
	t.Logf("PK round 1 RESULT: %s", resultMsg.Payload)

	// === Round 2: P1 eliminated ===

	// PK_START again
	assertMsgFromAll(t, conns, game.MsgPKStart)

	// Round 2 desc
	assertMsgFromAll(t, conns, game.MsgRound) // ROUND|pkNum|P0,P1
	assertMsgFromAll(t, conns, game.MsgTurn)  // TURN|P0
	fmt.Fprintf(conns[0], "DESC|P0 desc round 2\n")
	assertMsgFromAll(t, conns, game.MsgDesc)

	assertMsgFromAll(t, conns, game.MsgTurn) // TURN|P1
	fmt.Fprintf(conns[1], "DESC|P1 desc round 2\n")
	assertMsgFromAll(t, conns, game.MsgDesc)

	// PK voting round 2: P0→P1, P1→P0, P2→P1, P3→P1 → P0=1, P1=3 → P1 eliminated
	assertMsgFromAll(t, conns, game.MsgRound) // ROUND|pkNum|voterList
	assertMsgFromAll(t, conns, game.MsgTurn)  // TURN|P0
	fmt.Fprintf(conns[0], "PK_VOTE|P1\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	assertMsgFromAll(t, conns, game.MsgTurn) // TURN|P1
	fmt.Fprintf(conns[1], "PK_VOTE|P0\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	assertMsgFromAll(t, conns, game.MsgTurn) // TURN|P2
	fmt.Fprintf(conns[2], "PK_VOTE|P1\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	assertMsgFromAll(t, conns, game.MsgTurn) // TURN|P3
	fmt.Fprintf(conns[3], "PK_VOTE|P1\n")
	assertMsgFromAll(t, conns, game.MsgVoteBroadcast)

	// RESULT
	assertMsgFromAll(t, conns, game.MsgResult)

	// KICK|P1
	kickMsg := assertMsgFromAll(t, conns, game.MsgKick)
	if kickMsg.Payload != "P1" {
		t.Errorf("expected KICK P1, got %s", kickMsg.Payload)
	}

	eliminated := <-resultCh
	if eliminated != "P1" {
		t.Errorf("pkPhase returned %q, want %q", eliminated, "P1")
	}

	// Verify P1 is not alive
	for _, p := range players {
		if p.Name == "P1" && p.Alive {
			t.Error("P1 should be marked as not alive")
		}
	}
}

func TestDescriptionPhase_EmptyDescRejected(t *testing.T) {
	conns, cleanup := setupDescPhaseTest(t, 4)
	defer cleanup()

	// Read ROUND + TURN from all connections.
	var currentSpeaker string
	for i, conn := range conns {
		readMsgFromConn(t, conn) // ROUND
		msg := readMsgFromConn(t, conn) // TURN
		if i == 0 {
			currentSpeaker = msg.Payload
		}
	}

	// Find the current speaker's connection.
	speakerConn := findConnByName(conns, currentSpeaker)

	// Send empty description.
	fmt.Fprintf(speakerConn, "DESC|   \n")

	// Speaker should get ERROR.
	msg := readMsgFromConn(t, speakerConn)
	if msg.Type != game.MsgError {
		t.Errorf("expected ERROR for empty desc, got %s: %s", msg.Type, msg.Payload)
	}
	if !strings.Contains(msg.Payload, "描述不能为空") {
		t.Errorf("unexpected error payload: %q", msg.Payload)
	}

	// Send valid description to complete.
	fmt.Fprintf(speakerConn, "DESC|valid description\n")

	// All players get DESC broadcast.
	for _, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgDesc {
			t.Fatalf("expected DESC broadcast, got %s: %s", msg.Type, msg.Payload)
		}
	}

	// Then TURN for the next speaker.
	for _, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgTurn {
			t.Fatalf("expected TURN for next speaker, got %s: %s", msg.Type, msg.Payload)
		}
	}
}

func TestDescriptionPhase_NotYourTurn(t *testing.T) {
	conns, cleanup := setupDescPhaseTest(t, 4)
	defer cleanup()

	// Read ROUND + TURN from all connections.
	var currentSpeaker string
	for i, conn := range conns {
		readMsgFromConn(t, conn) // ROUND
		msg := readMsgFromConn(t, conn) // TURN
		if i == 0 {
			currentSpeaker = msg.Payload
		}
	}

	// Find a player who is NOT the current speaker and send DESC from them.
	for i, conn := range conns {
		name := fmt.Sprintf("P%d", i)
		if name != currentSpeaker {
			fmt.Fprintf(conn, "DESC|sneaky description\n")

			// Should get ERROR.
			msg := readMsgFromConn(t, conn)
			if msg.Type != game.MsgError {
				t.Errorf("expected ERROR for not-your-turn, got %s: %s", msg.Type, msg.Payload)
			}
			if !strings.Contains(msg.Payload, "还没轮到你") {
				t.Errorf("unexpected error payload: %q", msg.Payload)
			}
			return
		}
	}
}

// --- Judge quit mechanism tests ---

func TestNewStdinSource_QuitCommand(t *testing.T) {
	input := "hello\nquit\nworld\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	src := newStdinSource(scanner)

	// First line should be delivered via ch.
	select {
	case line := <-src.ch:
		if line != "hello" {
			t.Errorf("expected 'hello', got %q", line)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first line")
	}

	// 'quit' should close the quit channel and NOT send via ch.
	select {
	case <-src.quit:
		// expected
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for quit channel")
	}

	// done should also close since the goroutine exits.
	select {
	case <-src.done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for done channel")
	}

	// No more data should arrive on ch.
	select {
	case line := <-src.ch:
		t.Errorf("unexpected data on ch after quit: %q", line)
	default:
		// expected
	}
}

func TestNewStdinSource_EOF(t *testing.T) {
	input := "hello\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	src := newStdinSource(scanner)

	// First line delivered.
	select {
	case line := <-src.ch:
		if line != "hello" {
			t.Errorf("expected 'hello', got %q", line)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	// EOF should close done channel.
	select {
	case <-src.done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for done on EOF")
	}

	// quit should NOT be closed (EOF is not a quit command).
	select {
	case <-src.quit:
		t.Error("quit channel should not close on plain EOF")
	default:
		// expected
	}
}

func TestNewStdinSource_QuitCaseInsensitive(t *testing.T) {
	tests := []string{"quit", "QUIT", "Quit", "QuiT"}
	for _, quitInput := range tests {
		t.Run(quitInput, func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(quitInput + "\n"))
			src := newStdinSource(scanner)

			select {
			case <-src.quit:
				// expected
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for quit")
			}
		})
	}
}

func TestCollectWordsFromCh_QuitCh(t *testing.T) {
	out := &bytes.Buffer{}
	disp := newDisplay(out)
	stdinCh := make(chan string)
	stdinDone := make(chan struct{})
	quitCh := make(chan struct{})

	resultCh := make(chan struct{}, 1)
	go func() {
		civilian, undercover := collectWordsFromCh(out, disp, stdinCh, stdinDone, quitCh)
		if civilian == "" && undercover == "" {
			resultCh <- struct{}{}
		}
	}()

	// Trigger quit immediately.
	close(quitCh)

	select {
	case <-resultCh:
		// expected: collectWordsFromCh returned empty
	case <-time.After(time.Second):
		t.Fatal("collectWordsFromCh did not return on quit")
	}
}

func TestCollectWordsFromCh_StdinDone(t *testing.T) {
	out := &bytes.Buffer{}
	disp := newDisplay(out)
	stdinCh := make(chan string)
	stdinDone := make(chan struct{})
	quitCh := make(chan struct{})

	resultCh := make(chan struct{}, 1)
	go func() {
		civilian, undercover := collectWordsFromCh(out, disp, stdinCh, stdinDone, quitCh)
		if civilian == "" && undercover == "" {
			resultCh <- struct{}{}
		}
	}()

	// Trigger stdin done.
	close(stdinDone)

	select {
	case <-resultCh:
		// expected
	case <-time.After(time.Second):
		t.Fatal("collectWordsFromCh did not return on stdinDone")
	}
}

func TestRunJudge_QuitInWaitingPhase(t *testing.T) {
	// Config input followed by "quit" should cause RunJudge to return.
	configInput := "4\n1\n0\n"
	fullInput := configInput + "quit\n"

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen: %v", err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	out := &safeBuffer{}
	done := make(chan struct{})
	go func() {
		RunJudge(out, strings.NewReader(fullInput), port, false)
		close(done)
	}()

	select {
	case <-done:
		// expected: RunJudge returned
	case <-time.After(3 * time.Second):
		t.Fatal("RunJudge did not return after quit in waiting phase")
	}
}

func TestRunJudge_EOFInWaitingPhase(t *testing.T) {
	// Config input then EOF should cause RunJudge to return.
	configInput := "4\n1\n0\n"

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen: %v", err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	out := &safeBuffer{}
	done := make(chan struct{})
	go func() {
		RunJudge(out, strings.NewReader(configInput), port, false)
		close(done)
	}()

	select {
	case <-done:
		// expected
	case <-time.After(3 * time.Second):
		t.Fatal("RunJudge did not return after EOF in waiting phase")
	}
}

func TestRunJudge_QuitInCollectWords(t *testing.T) {
	// Config input + start + enough players + quit during word collection.
	// We use a pipe to simulate interactive stdin.
	configInput := "4\n1\n0\n"

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen: %v", err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	r, w := io.Pipe()
	out := &safeBuffer{}
	done := make(chan struct{})
	go func() {
		RunJudge(out, r, port, false)
		close(done)
	}()

	// Write config
	w.Write([]byte(configInput))

	// Wait for server to be ready
	for i := 0; i < 100; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 10*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Simulate 4 players joining
	for i := 0; i < 4; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 2*time.Second)
		if err != nil {
			w.Close()
			t.Fatalf("player %d connect failed: %v", i, err)
		}
		fmt.Fprintf(conn, "JOIN|P%d\n", i)
		// Keep connection alive
		_ = conn
	}

	// Wait for all players to be shown as joined
	waitForOutput(t, out, "P3 joined", 2*time.Second)

	// Send start
	w.Write([]byte("start\n"))

	// Wait for word prompt
	waitForOutput(t, out, "平民词语", 2*time.Second)

	// Send quit during word collection
	w.Write([]byte("quit\n"))

	select {
	case <-done:
		// expected: RunJudge returned after quit in collectWords
	case <-time.After(3 * time.Second):
		w.Close()
		t.Fatalf("RunJudge did not return after quit in collectWords, output: %s", out.String())
	}
	w.Close()
}

func TestDescriptionPhase_QuitCh(t *testing.T) {
	srv, conns, disp, cleanup := setupPKTestEnv(t, 4)
	defer cleanup()

	quitCh := make(chan struct{})
	resultCh := make(chan *descResult, 1)
	go func() {
		resultCh <- descriptionPhase(disp, srv, 1, []string{"P0", "P1", "P2", "P3"}, quitCh)
	}()

	// Consume ROUND + TURN messages so server channels are drained.
	assertMsgFromAll(t, conns, game.MsgRound)
	assertMsgFromAll(t, conns, game.MsgTurn)

	// Close quitCh to simulate referee quit.
	close(quitCh)

	select {
	case res := <-resultCh:
		if res != nil {
			t.Error("expected nil result on quit")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("descriptionPhase did not return on quit")
	}
}

func TestVotingPhase_QuitCh(t *testing.T) {
	srv, conns, disp, cleanup := setupPKTestEnv(t, 4)
	defer cleanup()

	players := []*game.Player{
		{Name: "P0", Alive: true},
		{Name: "P1", Alive: true},
		{Name: "P2", Alive: true},
		{Name: "P3", Alive: true},
	}

	quitCh := make(chan struct{})
	resultCh := make(chan string, 1)
	go func() {
		elim, _, _ := votingPhase(disp, srv, 1, players, quitCh)
		resultCh <- elim
	}()

	// Consume VOTE + TURN messages.
	assertMsgFromAll(t, conns, game.MsgVote)
	assertMsgFromAll(t, conns, game.MsgTurn)

	// Close quitCh to simulate referee quit.
	close(quitCh)

	select {
	case elim := <-resultCh:
		if elim != "" {
			t.Errorf("expected empty eliminated on quit, got %q", elim)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("votingPhase did not return on quit")
	}
}

func TestPKPhase_QuitCh(t *testing.T) {
	srv, conns, disp, cleanup := setupPKTestEnv(t, 4)
	defer cleanup()

	players := []*game.Player{
		{Name: "P0", Alive: true},
		{Name: "P1", Alive: true},
		{Name: "P2", Alive: true},
		{Name: "P3", Alive: true},
	}

	quitCh := make(chan struct{})
	resultCh := make(chan string, 1)
	go func() {
		resultCh <- pkPhase(disp, srv, []string{"P0", "P1"}, players, quitCh)
	}()

	// Consume PK_START + ROUND + TURN
	assertMsgFromAll(t, conns, game.MsgPKStart)
	assertMsgFromAll(t, conns, game.MsgRound)
	assertMsgFromAll(t, conns, game.MsgTurn)

	// Close quitCh to simulate referee quit.
	close(quitCh)

	select {
	case elim := <-resultCh:
		if elim != "" {
			t.Errorf("expected empty eliminated on quit, got %q", elim)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pkPhase did not return on quit")
	}
}

// --- Win condition unit tests ---

func TestCheckWinCondition_CiviliansWin(t *testing.T) {
	players := []*game.Player{
		{Name: "P0", Role: game.Civilian, Alive: true},
		{Name: "P1", Role: game.Civilian, Alive: true},
		{Name: "P2", Role: game.Undercover, Alive: false}, // eliminated
	}
	if got := game.CheckWinCondition(players); got != game.WinCivilians {
		t.Errorf("expected WinCivilians, got %q", got)
	}
}

func TestCheckWinCondition_UndercoversWin(t *testing.T) {
	// 1 civilian, 1 undercover alive → undercover wins
	players := []*game.Player{
		{Name: "P0", Role: game.Civilian, Alive: true},
		{Name: "P1", Role: game.Undercover, Alive: true},
		{Name: "P2", Role: game.Civilian, Alive: false},
	}
	if got := game.CheckWinCondition(players); got != game.WinUndercover {
		t.Errorf("expected WinUndercover, got %q", got)
	}
}

func TestCheckWinCondition_GameContinues(t *testing.T) {
	players := []*game.Player{
		{Name: "P0", Role: game.Civilian, Alive: true},
		{Name: "P1", Role: game.Civilian, Alive: true},
		{Name: "P2", Role: game.Undercover, Alive: true},
	}
	if got := game.CheckWinCondition(players); got != "" {
		t.Errorf("expected empty (game continues), got %q", got)
	}
}

func TestCheckWinCondition_BlankIgnored(t *testing.T) {
	// Blank doesn't count as civilian or undercover
	// 1 civilian, 1 undercover, 1 blank → undercover wins (civilian <= undercover)
	players := []*game.Player{
		{Name: "P0", Role: game.Civilian, Alive: true},
		{Name: "P1", Role: game.Undercover, Alive: true},
		{Name: "P2", Role: game.Blank, Alive: true},
	}
	if got := game.CheckWinCondition(players); got != game.WinUndercover {
		t.Errorf("expected WinUndercover, got %q", got)
	}
}

// --- Game loop integration tests ---

// runVotingRound runs one voting round: reads VOTE+TURN from all conns,
// has each voter vote for voteTarget, reads VOTE_BC from all conns.
func runVotingRound(t *testing.T, conns []net.Conn, numVoters int, voteTarget string) {
	t.Helper()
	// VOTE|roundNum|playerList broadcast to all
	assertMsgFromAll(t, conns, game.MsgVote)
	for i := 0; i < numVoters; i++ {
		// TURN|voterName broadcast to all
		turnMsg := assertMsgFromAll(t, conns, game.MsgTurn)
		voter := turnMsg.Payload
		voterConn := findConnByName(conns, voter)
		if voterConn == nil {
			t.Fatalf("cannot find connection for voter %q", voter)
		}
		fmt.Fprintf(voterConn, "VOTE|%s\n", voteTarget)
		// VOTE_BC broadcast to all
		assertMsgFromAll(t, conns, game.MsgVoteBroadcast)
	}
}

func TestGameLoop_CiviliansWin(t *testing.T) {
	_, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"P0", "P1", "P2", "P3"}
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	for _, conn := range conns {
		readMsgFromConn(t, conn) // JOIN
	}

	stdin <- "start"
	stdin <- "苹果"
	stdin <- "香蕉"

	// Read ROLE messages to identify who is undercover.
	undercoverName := ""
	civilianNames := make([]string, 0, 3)
	for i, conn := range conns {
		roleMsg := readMsgFromConn(t, conn)
		if roleMsg.Type != game.MsgRole {
			t.Fatalf("expected ROLE, got %s", roleMsg.Type)
		}
		parts := strings.SplitN(roleMsg.Payload, "|", 2)
		if parts[0] == "Undercover" {
			undercoverName = fmt.Sprintf("P%d", i)
		} else {
			civilianNames = append(civilianNames, fmt.Sprintf("P%d", i))
		}
		readMsgFromConn(t, conn) // READY
	}
	if undercoverName == "" {
		t.Fatal("no undercover found in ROLE messages")
	}
	t.Logf("undercover=%s, civilians=%v", undercoverName, civilianNames)

	// Strategy: eliminate 1 civilian then the undercover.
	// With 4 players (3C+1U): after eliminating 1C we have 2C+1U (continue),
	// after eliminating U we have 2C+0U → civilians win.
	eliminationOrder := []string{civilianNames[0], undercoverName}

	var savedRound *game.Message
	for round, voteTarget := range eliminationOrder {
		// Description phase.
		var roundMsg game.Message
		if savedRound != nil {
			roundMsg = *savedRound
			savedRound = nil
		} else {
			roundMsg = readMsgFromConn(t, conns[0])
		}
		if roundMsg.Type != game.MsgRound {
			t.Fatalf("round %d: expected ROUND, got %s", round+1, roundMsg.Type)
		}
		roundParts := strings.SplitN(roundMsg.Payload, "|", 2)
		speakers := strings.Split(roundParts[1], ",")
		for i := 1; i < len(conns); i++ {
			readMsgFromConn(t, conns[i]) // ROUND
		}
		assertMsgFromAll(t, conns, game.MsgTurn)

		for si, speaker := range speakers {
			speakerConn := findConnByName(conns, speaker)
			fmt.Fprintf(speakerConn, "DESC|desc from %s\n", speaker)
			assertMsgFromAll(t, conns, game.MsgDesc)
			if si < len(speakers)-1 {
				assertMsgFromAll(t, conns, game.MsgTurn)
			}
		}

		// Voting phase: vote to eliminate the target.
		runVotingRound(t, conns, len(speakers), voteTarget)

		// RESULT
		assertMsgFromAll(t, conns, game.MsgResult)
		// KICK
		kickMsg := assertMsgFromAll(t, conns, game.MsgKick)
		if kickMsg.Payload != voteTarget {
			t.Errorf("expected KICK %s, got %s", voteTarget, kickMsg.Payload)
		}

		// After KICK, the next message is either WIN or ROUND.
		nextMsg := readMsgFromConn(t, conns[0])
		if nextMsg.Type == game.MsgWin {
			if nextMsg.Payload != string(game.WinCivilians) {
				t.Errorf("expected civilian win, got %q", nextMsg.Payload)
			}
			for i := 1; i < len(conns); i++ {
				readMsgFromConn(t, conns[i]) // WIN
			}
			return // game over
		}
		// Not WIN — must be next round's ROUND. Save it.
		savedRound = &nextMsg
	}
	t.Fatal("game did not end after eliminating all players")
}

func TestGameLoop_RestartWithY(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"P0", "P1", "P2", "P3"}
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	for _, conn := range conns {
		readMsgFromConn(t, conn) // JOIN
	}

	stdin <- "start"
	stdin <- "苹果"
	stdin <- "香蕉"

	// Read ROLE messages to identify who is undercover.
	undercoverName := ""
	civilianNames := make([]string, 0, 3)
	for i, conn := range conns {
		roleMsg := readMsgFromConn(t, conn)
		if roleMsg.Type != game.MsgRole {
			t.Fatalf("expected ROLE, got %s", roleMsg.Type)
		}
		parts := strings.SplitN(roleMsg.Payload, "|", 2)
		if parts[0] == "Undercover" {
			undercoverName = fmt.Sprintf("P%d", i)
		} else {
			civilianNames = append(civilianNames, fmt.Sprintf("P%d", i))
		}
		readMsgFromConn(t, conn) // READY
	}
	if undercoverName == "" {
		t.Fatal("no undercover found in ROLE messages")
	}
	t.Logf("undercover=%s, civilians=%v", undercoverName, civilianNames)

	// Helper to run one description+voting round, returning the next message after KICK.
	// It reads the ROUND message from conns[0] internally.
	runGameRound := func(t *testing.T, voteTarget string) game.Message {
		t.Helper()
		roundMsg := readMsgFromConn(t, conns[0])
		if roundMsg.Type != game.MsgRound {
			t.Fatalf("expected ROUND, got %s", roundMsg.Type)
		}
		roundParts := strings.SplitN(roundMsg.Payload, "|", 2)
		speakers := strings.Split(roundParts[1], ",")
		for i := 1; i < len(conns); i++ {
			readMsgFromConn(t, conns[i]) // ROUND
		}
		assertMsgFromAll(t, conns, game.MsgTurn)

		for si, speaker := range speakers {
			speakerConn := findConnByName(conns, speaker)
			fmt.Fprintf(speakerConn, "DESC|desc\n")
			assertMsgFromAll(t, conns, game.MsgDesc)
			if si < len(speakers)-1 {
				assertMsgFromAll(t, conns, game.MsgTurn)
			}
		}

		runVotingRound(t, conns, len(speakers), voteTarget)
		assertMsgFromAll(t, conns, game.MsgResult)
		assertMsgFromAll(t, conns, game.MsgKick)

		nextMsg := readMsgFromConn(t, conns[0])
		return nextMsg
	}

	// === First game: civilian win ===
	// With 4 players (3C+1U), eliminate 1 civilian then the undercover.
	// After round 1: 2C+1U alive (continue).
	// After round 2: 2C+0U → civilians win.

	// Round 1: eliminate first civilian.
	nextMsg := runGameRound(t, civilianNames[0])
	if nextMsg.Type != game.MsgRound {
		t.Fatalf("expected ROUND after round 1, got %s", nextMsg.Type)
	}

	// Round 2: eliminate undercover → civilians win.
	// runGameRound reads ROUND from conns[0], but we already have it in nextMsg.
	// So inline this round.
	roundParts2 := strings.SplitN(nextMsg.Payload, "|", 2)
	speakers2 := strings.Split(roundParts2[1], ",")
	for i := 1; i < len(conns); i++ {
		readMsgFromConn(t, conns[i]) // ROUND
	}
	assertMsgFromAll(t, conns, game.MsgTurn)
	for si, speaker := range speakers2 {
		speakerConn := findConnByName(conns, speaker)
		fmt.Fprintf(speakerConn, "DESC|desc round2\n")
		assertMsgFromAll(t, conns, game.MsgDesc)
		if si < len(speakers2)-1 {
			assertMsgFromAll(t, conns, game.MsgTurn)
		}
	}
	runVotingRound(t, conns, len(speakers2), undercoverName)
	assertMsgFromAll(t, conns, game.MsgResult)
	assertMsgFromAll(t, conns, game.MsgKick)

	// WIN message should be broadcast.
	winMsg := assertMsgFromAll(t, conns, game.MsgWin)
	if winMsg.Payload != string(game.WinCivilians) {
		t.Errorf("expected civilian win, got %q", winMsg.Payload)
	}

	// Verify restart prompt appears in output.
	waitForOutput(t, out, "再来一局？(Y/N):", 2*time.Second)

	// Referee says Y.
	stdin <- "Y"

	// All clients should receive RESTART.
	assertMsgFromAll(t, conns, game.MsgRestart)

	// === Second game: referee inputs new words ===
	stdin <- "橘子"
	stdin <- "橙子"

	// All clients should receive new ROLE + READY for second game.
	type roleInfo struct {
		roleName string
		word     string
	}
	var roles []roleInfo
	for _, conn := range conns {
		roleMsg := readMsgFromConn(t, conn)
		if roleMsg.Type != game.MsgRole {
			t.Errorf("expected ROLE for second game, got %s", roleMsg.Type)
			continue
		}
		parts := strings.SplitN(roleMsg.Payload, "|", 2)
		if len(parts) < 2 {
			t.Errorf("malformed ROLE payload: %q", roleMsg.Payload)
			continue
		}
		roles = append(roles, roleInfo{parts[0], parts[1]})

		readyMsg := readMsgFromConn(t, conn)
		if readyMsg.Type != game.MsgReady {
			t.Errorf("expected READY for second game, got %s", readyMsg.Type)
		}
	}

	// Verify new words were assigned.
	for _, r := range roles {
		switch r.roleName {
		case "Civilian":
			if r.word != "橘子" {
				t.Errorf("second game Civilian got word %q, want %q", r.word, "橘子")
			}
		case "Undercover":
			if r.word != "橙子" {
				t.Errorf("second game Undercover got word %q, want %q", r.word, "橙子")
			}
		}
	}

	// Verify role distribution: 4 players, 1U → 3 civilians, 1 undercover
	civilianCount, undercoverCount := 0, 0
	for _, r := range roles {
		switch r.roleName {
		case "Civilian":
			civilianCount++
		case "Undercover":
			undercoverCount++
		}
	}
	if civilianCount != 3 {
		t.Errorf("expected 3 civilians in second game, got %d", civilianCount)
	}
	if undercoverCount != 1 {
		t.Errorf("expected 1 undercover in second game, got %d", undercoverCount)
	}

	// Second game: identify undercover and play one round.
	secondGameUndercover := ""
	secondGameCivilians := make([]string, 0, 3)
	for i, r := range roles {
		pname := fmt.Sprintf("P%d", i)
		if r.roleName == "Undercover" {
			secondGameUndercover = pname
		} else {
			secondGameCivilians = append(secondGameCivilians, pname)
		}
	}
	t.Logf("second game: undercover=%s, civilians=%v", secondGameUndercover, secondGameCivilians)

	// Play one round: eliminate a civilian so the game continues.
	roundMsg4 := readMsgFromConn(t, conns[0])
	if roundMsg4.Type != game.MsgRound {
		t.Fatalf("expected ROUND in second game, got %s", roundMsg4.Type)
	}
	roundParts4 := strings.SplitN(roundMsg4.Payload, "|", 2)
	speakers4 := strings.Split(roundParts4[1], ",")
	for i := 1; i < len(conns); i++ {
		readMsgFromConn(t, conns[i]) // ROUND
	}
	assertMsgFromAll(t, conns, game.MsgTurn)
	for si, speaker := range speakers4 {
		speakerConn := findConnByName(conns, speaker)
		fmt.Fprintf(speakerConn, "DESC|desc second game\n")
		assertMsgFromAll(t, conns, game.MsgDesc)
		if si < len(speakers4)-1 {
			assertMsgFromAll(t, conns, game.MsgTurn)
		}
	}

	// Vote to eliminate a civilian (not the undercover) so game continues.
	voteTarget4 := secondGameCivilians[0]
	// Make sure voteTarget4 is among the speakers.
	found := false
	for _, s := range speakers4 {
		if s == voteTarget4 {
			found = true
			break
		}
	}
	if !found {
		// Fallback: use first speaker that is a civilian.
		for _, s := range speakers4 {
			if s != secondGameUndercover {
				voteTarget4 = s
				break
			}
		}
	}
	runVotingRound(t, conns, len(speakers4), voteTarget4)
	assertMsgFromAll(t, conns, game.MsgResult)
	assertMsgFromAll(t, conns, game.MsgKick)

	// Verify "新一局即将开始" message appears (already printed after Y input).
	waitForOutput(t, out, "新一局即将开始", 2*time.Second)
}

func TestGameLoop_ShutdownWithN(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"P0", "P1", "P2", "P3"}
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	for _, conn := range conns {
		readMsgFromConn(t, conn) // JOIN
	}

	stdin <- "start"
	stdin <- "苹果"
	stdin <- "香蕉"

	// Read ROLE messages to identify who is undercover.
	undercoverName := ""
	civilianNames := make([]string, 0, 3)
	for i, conn := range conns {
		roleMsg := readMsgFromConn(t, conn)
		if roleMsg.Type != game.MsgRole {
			t.Fatalf("expected ROLE, got %s", roleMsg.Type)
		}
		parts := strings.SplitN(roleMsg.Payload, "|", 2)
		if parts[0] == "Undercover" {
			undercoverName = fmt.Sprintf("P%d", i)
		} else {
			civilianNames = append(civilianNames, fmt.Sprintf("P%d", i))
		}
		readMsgFromConn(t, conn) // READY
	}
	if undercoverName == "" {
		t.Fatal("no undercover found in ROLE messages")
	}

	// Eliminate 1 civilian then the undercover for civilian win (2 rounds).
	// With 4 players (3C+1U): after round 1 → 2C+1U (continue), after round 2 → 2C+0U (civilians win).

	// Helper: run one desc+voting round given a pre-read ROUND message.
	// Returns the next message after KICK from conns[0].
	runRoundFromMsg := func(roundMsg game.Message, voteTarget string) game.Message {
		roundParts := strings.SplitN(roundMsg.Payload, "|", 2)
		speakers := strings.Split(roundParts[1], ",")
		for i := 1; i < len(conns); i++ {
			readMsgFromConn(t, conns[i])
		}
		assertMsgFromAll(t, conns, game.MsgTurn)
		for si, speaker := range speakers {
			speakerConn := findConnByName(conns, speaker)
			fmt.Fprintf(speakerConn, "DESC|desc\n")
			assertMsgFromAll(t, conns, game.MsgDesc)
			if si < len(speakers)-1 {
				assertMsgFromAll(t, conns, game.MsgTurn)
			}
		}
		runVotingRound(t, conns, len(speakers), voteTarget)
		assertMsgFromAll(t, conns, game.MsgResult)
		assertMsgFromAll(t, conns, game.MsgKick)
		return readMsgFromConn(t, conns[0])
	}

	// Round 1: eliminate a civilian.
	roundMsg := readMsgFromConn(t, conns[0])
	nextMsg := runRoundFromMsg(roundMsg, civilianNames[0])

	// Round 2: eliminate undercover → civilians win.
	// Inline this round so we don't consume WIN from conns[0] inside runRoundFromMsg.
	roundParts2 := strings.SplitN(nextMsg.Payload, "|", 2)
	speakers2 := strings.Split(roundParts2[1], ",")
	for i := 1; i < len(conns); i++ {
		readMsgFromConn(t, conns[i])
	}
	assertMsgFromAll(t, conns, game.MsgTurn)
	for si, speaker := range speakers2 {
		speakerConn := findConnByName(conns, speaker)
		fmt.Fprintf(speakerConn, "DESC|desc\n")
		assertMsgFromAll(t, conns, game.MsgDesc)
		if si < len(speakers2)-1 {
			assertMsgFromAll(t, conns, game.MsgTurn)
		}
	}
	runVotingRound(t, conns, len(speakers2), undercoverName)
	assertMsgFromAll(t, conns, game.MsgResult)
	assertMsgFromAll(t, conns, game.MsgKick)

	// WIN
	winMsg := assertMsgFromAll(t, conns, game.MsgWin)
	if winMsg.Payload != string(game.WinCivilians) {
		t.Errorf("expected civilian win, got %q", winMsg.Payload)
	}

	// Verify restart prompt.
	waitForOutput(t, out, "再来一局？(Y/N):", 2*time.Second)

	// Referee says N.
	stdin <- "N"

	// All clients should receive QUIT broadcast.
	quitMsg := assertMsgFromAll(t, conns, game.MsgQuit)
	if quitMsg.Payload != "裁判结束了游戏" {
		t.Errorf("expected QUIT payload '裁判结束了游戏', got %q", quitMsg.Payload)
	}
}

func TestPlayerHandleMsg_Restart(t *testing.T) {
	out := &bytes.Buffer{}
	disp := client.NewDisplay(out, false)
	phase := descSubmitted
	inPK := true
	inVoteRound := true
	pkSpeakers := []string{"P0", "P1"}

	msg := game.Message{Type: game.MsgRestart}
	ok := handleMessage(msg, disp, out, nil, "P0", &phase, &inPK, &inVoteRound, &pkSpeakers)
	if !ok {
		t.Error("handleMessage should return true for RESTART")
	}
	if phase != descIdle {
		t.Errorf("expected descIdle after RESTART, got %d", phase)
	}
	if inPK {
		t.Error("expected inPK=false after RESTART")
	}
	if inVoteRound {
		t.Error("expected inVoteRound=false after RESTART")
	}
	if pkSpeakers != nil {
		t.Error("expected pkSpeakers=nil after RESTART")
	}
	if !strings.Contains(out.String(), "新一局即将开始") {
		t.Errorf("expected restart message, got: %s", out.String())
	}
}

func TestPlayerHandleMsg_VoteBroadcast(t *testing.T) {
	out := &bytes.Buffer{}
	disp := client.NewDisplay(out, false)
	phase := voteSubmitted

	msg := game.Message{Type: game.MsgVoteBroadcast, Payload: "P0|P2"}
	ok := handleMessage(msg, disp, out, nil, "P1", &phase, nil, nil, nil)
	if !ok {
		t.Error("handleMessage should return true for VOTE_BC")
	}
	if phase != descIdle {
		t.Errorf("expected descIdle after VOTE_BC, got %d", phase)
	}
	if !strings.Contains(out.String(), "P0 投票给了 P2") {
		t.Errorf("expected vote broadcast display, got: %s", out.String())
	}
}

// playGameToWinCondition is a test helper that plays a 2-round game with 4 players
// (3C+1U) to reach a civilian win. It returns the undercover and civilian names.
func playGameToWinCondition(t *testing.T, conns []net.Conn, stdin chan string, out *safeBuffer) (string, []string) {
	t.Helper()
	stdin <- "start"
	stdin <- "苹果"
	stdin <- "香蕉"

	undercoverName := ""
	civilianNames := make([]string, 0, 3)
	for i, conn := range conns {
		roleMsg := readMsgFromConn(t, conn)
		if roleMsg.Type != game.MsgRole {
			t.Fatalf("expected ROLE, got %s", roleMsg.Type)
		}
		parts := strings.SplitN(roleMsg.Payload, "|", 2)
		if parts[0] == "Undercover" {
			undercoverName = fmt.Sprintf("P%d", i)
		} else {
			civilianNames = append(civilianNames, fmt.Sprintf("P%d", i))
		}
		readMsgFromConn(t, conn) // READY
	}
	if undercoverName == "" {
		t.Fatal("no undercover found")
	}

	// Round 1: eliminate first civilian.
	roundMsg := readMsgFromConn(t, conns[0])
	roundParts := strings.SplitN(roundMsg.Payload, "|", 2)
	speakers := strings.Split(roundParts[1], ",")
	for i := 1; i < len(conns); i++ {
		readMsgFromConn(t, conns[i])
	}
	assertMsgFromAll(t, conns, game.MsgTurn)
	for si, speaker := range speakers {
		speakerConn := findConnByName(conns, speaker)
		fmt.Fprintf(speakerConn, "DESC|desc\n")
		assertMsgFromAll(t, conns, game.MsgDesc)
		if si < len(speakers)-1 {
			assertMsgFromAll(t, conns, game.MsgTurn)
		}
	}
	runVotingRound(t, conns, len(speakers), civilianNames[0])
	assertMsgFromAll(t, conns, game.MsgResult)
	assertMsgFromAll(t, conns, game.MsgKick)

	// Round 2: eliminate undercover → civilians win.
	nextMsg := readMsgFromConn(t, conns[0])
	roundParts2 := strings.SplitN(nextMsg.Payload, "|", 2)
	speakers2 := strings.Split(roundParts2[1], ",")
	for i := 1; i < len(conns); i++ {
		readMsgFromConn(t, conns[i])
	}
	assertMsgFromAll(t, conns, game.MsgTurn)
	for si, speaker := range speakers2 {
		speakerConn := findConnByName(conns, speaker)
		fmt.Fprintf(speakerConn, "DESC|desc2\n")
		assertMsgFromAll(t, conns, game.MsgDesc)
		if si < len(speakers2)-1 {
			assertMsgFromAll(t, conns, game.MsgTurn)
		}
	}
	runVotingRound(t, conns, len(speakers2), undercoverName)
	assertMsgFromAll(t, conns, game.MsgResult)
	assertMsgFromAll(t, conns, game.MsgKick)

	assertMsgFromAll(t, conns, game.MsgWin)
	waitForOutput(t, out, "再来一局？(Y/N):", 2*time.Second)

	return undercoverName, civilianNames
}

func connectPlayers(t *testing.T, port string, names []string) []net.Conn {
	t.Helper()
	conns := make([]net.Conn, len(names))
	for i, name := range names {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}
	for _, conn := range conns {
		readMsgFromConn(t, conn) // JOIN ack
	}
	return conns
}

func TestGameLoop_RestartWithInvalidInput(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"P0", "P1", "P2", "P3"}
	conns := connectPlayers(t, port, names)
	for i := range conns {
		defer conns[i].Close()
	}

	playGameToWinCondition(t, conns, stdin, out)

	// Input an invalid string (neither Y nor y).
	stdin <- "yes"

	// Should behave like N: broadcast QUIT and stop.
	quitMsg := assertMsgFromAll(t, conns, game.MsgQuit)
	if quitMsg.Payload != "裁判结束了游戏" {
		t.Errorf("expected QUIT for invalid input, got %q", quitMsg.Payload)
	}
}

func TestGameLoop_RestartWithLowercaseY(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"P0", "P1", "P2", "P3"}
	conns := connectPlayers(t, port, names)
	for i := range conns {
		defer conns[i].Close()
	}

	playGameToWinCondition(t, conns, stdin, out)

	// Input lowercase "y" — should trigger restart.
	stdin <- "y"

	// All clients should receive RESTART.
	assertMsgFromAll(t, conns, game.MsgRestart)

	// Second game: referee inputs new words.
	stdin <- "橘子"
	stdin <- "橙子"

	// Verify new ROLE messages arrive.
	for _, conn := range conns {
		roleMsg := readMsgFromConn(t, conn)
		if roleMsg.Type != game.MsgRole {
			t.Fatalf("expected ROLE for second game, got %s", roleMsg.Type)
		}
		readMsgFromConn(t, conn) // READY
	}
}

func TestGameLoop_RestartWithEmptyInput(t *testing.T) {
	out, port, stdin, cleanup := startJudgeForTestWithStdin(t, "4\n1\n0\n")
	defer cleanup()

	names := []string{"P0", "P1", "P2", "P3"}
	conns := connectPlayers(t, port, names)
	for i := range conns {
		defer conns[i].Close()
	}

	playGameToWinCondition(t, conns, stdin, out)

	// Input empty line — should behave like N.
	stdin <- ""

	quitMsg := assertMsgFromAll(t, conns, game.MsgQuit)
	if quitMsg.Payload != "裁判结束了游戏" {
		t.Errorf("expected QUIT for empty input, got %q", quitMsg.Payload)
	}
}
