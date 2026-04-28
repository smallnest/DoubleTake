package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/smallnest/doubletake/client"
	"github.com/smallnest/doubletake/game"
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
