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

// --- Voting phase integration tests ---

// setupVotePhaseTest is a test helper that sets up a game through the description phase
// and returns the player connections, the voter order (same as speaker order in desc phase),
// and a cleanup function. After this returns, all connections are ready for the voting phase.
func setupVotePhaseTest(t *testing.T, numPlayers int) ([]net.Conn, []string, func()) {
	t.Helper()

	conns, cleanup := setupDescPhaseTest(t, numPlayers)

	// Read ROUND + TURN from all connections to learn speaker order.
	var speakers []string
	for i, conn := range conns {
		roundMsg := readMsgFromConn(t, conn)
		if roundMsg.Type != game.MsgRound {
			t.Fatalf("conn %d: expected ROUND, got %s", i, roundMsg.Type)
		}
		roundParts := strings.SplitN(roundMsg.Payload, "|", 2)
		speakers = strings.Split(roundParts[1], ",")

		turnMsg := readMsgFromConn(t, conn)
		if turnMsg.Type != game.MsgTurn {
			t.Fatalf("conn %d: expected TURN, got %s", i, turnMsg.Type)
		}
	}

	// Each speaker describes in order to complete the description phase.
	for _, speaker := range speakers {
		speakerConn := findConnByName(conns, speaker)
		fmt.Fprintf(speakerConn, "DESC|hello from %s\n", speaker)

		// All players receive DESC broadcast.
		for _, conn := range conns {
			readMsgFromConn(t, conn)
		}

		// After DESC, TURN for next speaker (except last).
		if speaker != speakers[len(speakers)-1] {
			for _, conn := range conns {
				readMsgFromConn(t, conn)
			}
		}
	}

	return conns, speakers, cleanup
}

func TestVotingPhase_NormalFlow(t *testing.T) {
	conns, voters, cleanup := setupVotePhaseTest(t, 4)
	defer cleanup()

	// All players should receive VOTE|1|P0,P1,P2,P3.
	voteMsg := readMsgFromConn(t, conns[0])
	if voteMsg.Type != game.MsgVote {
		t.Fatalf("expected VOTE, got %s: %s", voteMsg.Type, voteMsg.Payload)
	}
	voteParts := strings.SplitN(voteMsg.Payload, "|", 2)
	if len(voteParts) < 2 {
		t.Fatalf("malformed VOTE payload: %q", voteMsg.Payload)
	}
	if voteParts[0] != "1" {
		t.Errorf("expected round 1, got %q", voteParts[0])
	}

	// All players should receive TURN for the first voter.
	turnMsg := readMsgFromConn(t, conns[0])
	if turnMsg.Type != game.MsgTurn {
		t.Fatalf("expected TURN, got %s: %s", turnMsg.Type, turnMsg.Payload)
	}

	// Consume VOTE + TURN from remaining connections.
	for i := 1; i < len(conns); i++ {
		msg := readMsgFromConn(t, conns[i])
		if msg.Type != game.MsgVote {
			t.Errorf("conn %d: expected VOTE, got %s", i, msg.Type)
		}
		msg = readMsgFromConn(t, conns[i])
		if msg.Type != game.MsgTurn {
			t.Errorf("conn %d: expected TURN, got %s", i, msg.Type)
		}
	}

	// Each voter votes in order. All vote for voters[0] to ensure a clear elimination
	// (nobody votes for themselves since voters[0] votes for voters[1] and others vote for voters[0]).
	for vi, voter := range voters {
		var target string
		if vi == 0 {
			target = voters[1] // voters[0] votes for voters[1]
		} else {
			target = voters[0] // everyone else votes for voters[0]
		}
		voterConn := findConnByName(conns, voter)
		fmt.Fprintf(voterConn, "VOTE|%s\n", target)

		// After voting, next voter gets TURN (unless this is the last voter).
		if vi < len(voters)-1 {
			for _, conn := range conns {
				msg := readMsgFromConn(t, conn)
				if msg.Type != game.MsgTurn {
					t.Fatalf("expected TURN for next voter, got %s: %s", msg.Type, msg.Payload)
				}
			}
		}
	}

	// All players should receive RESULT with vote tallies.
	for i, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgResult {
			t.Fatalf("conn %d: expected RESULT, got %s: %s", i, msg.Type, msg.Payload)
		}
		// voters[0] gets 3 votes (from voters[1], voters[2], voters[3]).
		if !strings.Contains(msg.Payload, voters[0]+":3") {
			t.Errorf("conn %d: RESULT payload should contain %s:3, got %q", i, voters[0], msg.Payload)
		}
	}
}

func TestVotingPhase_ValidationRejectsInvalidVote(t *testing.T) {
	conns, voters, cleanup := setupVotePhaseTest(t, 4)
	defer cleanup()

	// Consume VOTE + TURN from all connections.
	var currentVoter string
	for i, conn := range conns {
		readMsgFromConn(t, conn) // VOTE
		msg := readMsgFromConn(t, conn) // TURN
		if i == 0 {
			currentVoter = msg.Payload
		}
	}

	voterConn := findConnByName(conns, currentVoter)

	// Send empty vote — should get ERROR.
	fmt.Fprintf(voterConn, "VOTE|   \n")
	msg := readMsgFromConn(t, voterConn)
	if msg.Type != game.MsgError {
		t.Fatalf("expected ERROR for empty vote, got %s: %s", msg.Type, msg.Payload)
	}
	if !strings.Contains(msg.Payload, "vote target must not be empty") {
		t.Errorf("unexpected error payload: %q", msg.Payload)
	}

	// Send self-vote — should get ERROR.
	fmt.Fprintf(voterConn, "VOTE|%s\n", currentVoter)
	msg = readMsgFromConn(t, voterConn)
	if msg.Type != game.MsgError {
		t.Fatalf("expected ERROR for self-vote, got %s: %s", msg.Type, msg.Payload)
	}
	if !strings.Contains(msg.Payload, "cannot vote for yourself") {
		t.Errorf("unexpected error payload: %q", msg.Payload)
	}

	// Send valid vote to proceed.
	target := voters[0]
	if target == currentVoter {
		target = voters[1]
	}
	fmt.Fprintf(voterConn, "VOTE|%s\n", target)

	// Next voter gets TURN.
	for _, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgTurn {
			t.Fatalf("expected TURN for next voter, got %s: %s", msg.Type, msg.Payload)
		}
	}
}

func TestVotingPhase_NotYourTurn(t *testing.T) {
	conns, voters, cleanup := setupVotePhaseTest(t, 4)
	defer cleanup()

	// Consume VOTE + TURN from all connections.
	var currentVoter string
	for i, conn := range conns {
		readMsgFromConn(t, conn) // VOTE
		msg := readMsgFromConn(t, conn) // TURN
		if i == 0 {
			currentVoter = msg.Payload
		}
	}

	// Find a player who is NOT the current voter and send VOTE from them.
	for i, conn := range conns {
		name := fmt.Sprintf("P%d", i)
		if name != currentVoter {
			fmt.Fprintf(conn, "VOTE|%s\n", voters[0])

			// Should get ERROR.
			msg := readMsgFromConn(t, conn)
			if msg.Type != game.MsgError {
				t.Errorf("expected ERROR for not-your-turn, got %s: %s", msg.Type, msg.Payload)
			}
			if !strings.Contains(msg.Payload, "还没轮到你投票") {
				t.Errorf("unexpected error payload: %q", msg.Payload)
			}
			return
		}
	}
}

func TestVotingPhase_ResultBroadcastAndElimination(t *testing.T) {
	conns, voters, cleanup := setupVotePhaseTest(t, 4)
	defer cleanup()

	// Consume VOTE + TURN from all connections.
	for _, conn := range conns {
		readMsgFromConn(t, conn) // VOTE
		readMsgFromConn(t, conn) // TURN
	}

	// All voters vote for voters[0] except voters[0] who votes for voters[1].
	// This ensures voters[0] gets 3 votes and is eliminated.
	target := voters[0]
	for vi, voter := range voters {
		var voteTarget string
		if vi == 0 {
			voteTarget = voters[1]
		} else {
			voteTarget = target
		}
		voterConn := findConnByName(conns, voter)
		fmt.Fprintf(voterConn, "VOTE|%s\n", voteTarget)

		if vi < len(voters)-1 {
			for _, conn := range conns {
				readMsgFromConn(t, conn) // TURN for next voter
			}
		}
	}

	// Read RESULT from first connection and verify.
	resultMsg := readMsgFromConn(t, conns[0])
	if resultMsg.Type != game.MsgResult {
		t.Fatalf("expected RESULT, got %s: %s", resultMsg.Type, resultMsg.Payload)
	}
	// Result should show target:3 (3 voters voted for voters[0]).
	if !strings.Contains(resultMsg.Payload, target+":3") {
		t.Errorf("RESULT should show %s:3, got %q", target, resultMsg.Payload)
	}

	// Read RESULT from remaining connections (it's a broadcast).
	for i := 1; i < len(conns); i++ {
		msg := readMsgFromConn(t, conns[i])
		if msg.Type != game.MsgResult {
			t.Errorf("conn %d: expected RESULT, got %s", i, msg.Type)
		}
	}
}

// --- PK phase integration tests ---

// doVoting is a test helper that drives the voting phase given a vote mapping.
// It reads VOTE+TURN from all conns, sends votes in order, reads TURN for subsequent
// voters, and reads the RESULT broadcast. Returns the RESULT message from conns[0].
func doVoting(t *testing.T, conns []net.Conn, voters []string, votes map[string]string) game.Message {
	t.Helper()

	// Read VOTE + TURN from all connections.
	for _, conn := range conns {
		readMsgFromConn(t, conn) // VOTE
		readMsgFromConn(t, conn) // TURN
	}

	for vi, voter := range voters {
		voterConn := findConnByName(conns, voter)
		fmt.Fprintf(voterConn, "VOTE|%s\n", votes[voter])

		if vi < len(voters)-1 {
			for _, conn := range conns {
				readMsgFromConn(t, conn) // TURN for next voter
			}
		}
	}

	// Read RESULT from all connections.
	resultMsg := readMsgFromConn(t, conns[0])
	if resultMsg.Type != game.MsgResult {
		t.Fatalf("expected RESULT, got %s: %s", resultMsg.Type, resultMsg.Payload)
	}
	for i := 1; i < len(conns); i++ {
		readMsgFromConn(t, conns[i]) // RESULT
	}
	return resultMsg
}

func TestPKPhase_TieTriggersPK(t *testing.T) {
	conns, voters, cleanup := setupVotePhaseTest(t, 4)
	defer cleanup()

	// Create a tie: P0→P1, P1→P0, P2→P0, P3→P1 → P0:2, P1:2
	votes := map[string]string{
		voters[0]: voters[1],
		voters[1]: voters[0],
		voters[2]: voters[0],
		voters[3]: voters[1],
	}
	resultMsg := doVoting(t, conns, voters, votes)

	// Verify tie in RESULT.
	if !strings.Contains(resultMsg.Payload, voters[0]+":2") {
		t.Errorf("RESULT should contain %s:2, got %q", voters[0], resultMsg.Payload)
	}
	if !strings.Contains(resultMsg.Payload, voters[1]+":2") {
		t.Errorf("RESULT should contain %s:2, got %q", voters[1], resultMsg.Payload)
	}

	// All connections should receive PK_START.
	for i, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgPKStart {
			t.Fatalf("conn %d: expected PK_START, got %s: %s", i, msg.Type, msg.Payload)
		}
		// PK_START payload: "1|P0,P1" (or P1,P0)
		if !strings.Contains(msg.Payload, voters[0]) || !strings.Contains(msg.Payload, voters[1]) {
			t.Errorf("PK_START should contain both tied players, got %q", msg.Payload)
		}
	}
}

func TestPKPhase_FullFlow(t *testing.T) {
	conns, voters, cleanup := setupVotePhaseTest(t, 4)
	defer cleanup()

	// Create a tie: P0→P1, P1→P0, P2→P0, P3→P1 → P0:2, P1:2
	votes := map[string]string{
		voters[0]: voters[1],
		voters[1]: voters[0],
		voters[2]: voters[0],
		voters[3]: voters[1],
	}
	doVoting(t, conns, voters, votes)

	// Read PK_START from all connections.
	for _, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgPKStart {
			t.Fatalf("expected PK_START, got %s", msg.Type)
		}
	}

	// Read TURN for first PK speaker from all connections.
	var firstSpeaker string
	for i, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgTurn {
			t.Fatalf("conn %d: expected TURN, got %s", i, msg.Type)
		}
		if i == 0 {
			firstSpeaker = msg.Payload
		}
	}

	// Identify tied players from PK_START (firstSpeaker is one of them).
	// The other tied player is the one who's not firstSpeaker.
	var otherTied string
	if firstSpeaker == voters[0] {
		otherTied = voters[1]
	} else {
		otherTied = voters[0]
	}

	// First tied player describes.
	firstConn := findConnByName(conns, firstSpeaker)
	fmt.Fprintf(firstConn, "DESC|PK desc from %s\n", firstSpeaker)

	// All connections receive DESC broadcast.
	for _, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgDesc {
			t.Fatalf("expected DESC, got %s: %s", msg.Type, msg.Payload)
		}
	}

	// All connections receive TURN for second tied player.
	for _, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgTurn {
			t.Fatalf("expected TURN, got %s: %s", msg.Type, msg.Payload)
		}
		if msg.Payload != otherTied {
			t.Errorf("expected TURN for %s, got %s", otherTied, msg.Payload)
		}
	}

	// Second tied player describes.
	otherConn := findConnByName(conns, otherTied)
	fmt.Fprintf(otherConn, "DESC|PK desc from %s\n", otherTied)

	// All connections receive DESC broadcast.
	for _, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgDesc {
			t.Fatalf("expected DESC, got %s: %s", msg.Type, msg.Payload)
		}
	}

	// All connections receive PK_VOTE.
	for i, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgPKVote {
			t.Fatalf("conn %d: expected PK_VOTE, got %s: %s", i, msg.Type, msg.Payload)
		}
	}

	// All connections receive TURN for first PK voter.
	var firstPKVoter string
	for i, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgTurn {
			t.Fatalf("conn %d: expected TURN, got %s", i, msg.Type)
		}
		if i == 0 {
			firstPKVoter = msg.Payload
		}
	}

	// Vote in PK: discover voter order from TURN messages.
	// Everyone votes for firstSpeaker (one of the tied players) to ensure clear
	// elimination, but that tied player votes for otherTied (can't vote for self).
	// We use firstPKVoter (from TURN) to start, not firstSpeaker (from desc order).
	numPKVoters := len(voters)
	nextVoter := firstPKVoter
	for i := 0; i < numPKVoters; i++ {
		voterConn := findConnByName(conns, nextVoter)
		var target string
		if nextVoter == firstSpeaker {
			target = otherTied
		} else {
			target = firstSpeaker
		}
		fmt.Fprintf(voterConn, "VOTE|%s\n", target)

		if i < numPKVoters-1 {
			// Read TURN for next voter from all connections.
			turnMsg := readMsgFromConn(t, conns[0])
			if turnMsg.Type != game.MsgTurn {
				t.Fatalf("expected TURN for next PK voter, got %s: %s", turnMsg.Type, turnMsg.Payload)
			}
			nextVoter = turnMsg.Payload
			for j := 1; j < len(conns); j++ {
				readMsgFromConn(t, conns[j])
			}
		}
	}

	// Read PK RESULT from all connections.
	for i, conn := range conns {
		msg := readMsgFromConn(t, conn)
		if msg.Type != game.MsgResult {
			t.Fatalf("conn %d: expected RESULT, got %s: %s", i, msg.Type, msg.Payload)
		}
		// firstSpeaker should have 3 votes (from all except themselves).
		if !strings.Contains(msg.Payload, firstSpeaker+":3") {
			t.Errorf("conn %d: RESULT should show %s:3, got %q", i, firstSpeaker, msg.Payload)
		}
	}
}

func TestPKPhase_InvalidVoteForNonTiedPlayer(t *testing.T) {
	conns, voters, cleanup := setupVotePhaseTest(t, 4)
	defer cleanup()

	// Create a tie: P0→P1, P1→P0, P2→P0, P3→P1 → P0:2, P1:2
	votes := map[string]string{
		voters[0]: voters[1],
		voters[1]: voters[0],
		voters[2]: voters[0],
		voters[3]: voters[1],
	}
	doVoting(t, conns, voters, votes)

	// Read PK_START.
	for _, conn := range conns {
		readMsgFromConn(t, conn) // PK_START
	}

	// Read TURN for first PK speaker.
	var firstSpeaker string
	for i, conn := range conns {
		msg := readMsgFromConn(t, conn) // TURN
		if i == 0 {
			firstSpeaker = msg.Payload
		}
	}

	var otherTied string
	if firstSpeaker == voters[0] {
		otherTied = voters[1]
	} else {
		otherTied = voters[0]
	}

	// Both tied players describe.
	firstConn := findConnByName(conns, firstSpeaker)
	fmt.Fprintf(firstConn, "DESC|desc1\n")
	for _, conn := range conns {
		readMsgFromConn(t, conn) // DESC
	}
	for _, conn := range conns {
		readMsgFromConn(t, conn) // TURN
	}

	otherConn := findConnByName(conns, otherTied)
	fmt.Fprintf(otherConn, "DESC|desc2\n")
	for _, conn := range conns {
		readMsgFromConn(t, conn) // DESC
	}

	// Read PK_VOTE + TURN from all.
	var firstPKVoter string
	for i, conn := range conns {
		readMsgFromConn(t, conn) // PK_VOTE
		msg := readMsgFromConn(t, conn) // TURN
		if i == 0 {
			firstPKVoter = msg.Payload
		}
	}

	// First PK voter tries to vote for a non-tied player.
	// Find a non-tied player who is NOT the firstPKVoter (to avoid self-vote check).
	var nonTiedTarget string
	for _, v := range voters {
		if v != firstSpeaker && v != otherTied && v != firstPKVoter {
			nonTiedTarget = v
			break
		}
	}
	if nonTiedTarget == "" {
		t.Fatal("could not find a non-tied target different from firstPKVoter")
	}
	firstVoterConn := findConnByName(conns, firstPKVoter)

	// Try voting for a non-tied player — should get ERROR.
	fmt.Fprintf(firstVoterConn, "VOTE|%s\n", nonTiedTarget)
	msg := readMsgFromConn(t, firstVoterConn)
	if msg.Type != game.MsgError {
		t.Fatalf("expected ERROR for non-tied vote, got %s: %s", msg.Type, msg.Payload)
	}
	if !strings.Contains(msg.Payload, "tied players") {
		t.Errorf("error should mention tied players, got: %q", msg.Payload)
	}
}

// --- Game loop and WIN message integration tests ---

// setupGameLoopTest is a test helper that sets up a game through the ready phase
// and returns the player connections, speaker order, and cleanup function.
// After this returns, all connections have consumed JOIN, ROLE, and READY messages,
// and are ready for the first ROUND message.
func setupGameLoopTest(t *testing.T, numPlayers int) ([]net.Conn, []string, func()) {
	t.Helper()

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
		readMsgFromConn(t, conn)
	}

	stdin <- "start"
	stdin <- "苹果"
	stdin <- "香蕉"

	// Consume ROLE + READY from each connection.
	for _, conn := range conns {
		readMsgFromConn(t, conn) // ROLE
		readMsgFromConn(t, conn) // READY
	}

	waitForOutput(t, out, "游戏开始！", 2*time.Second)

	// Read ROUND + TURN from all connections to learn speaker order.
	var speakers []string
	for i, conn := range conns {
		roundMsg := readMsgFromConn(t, conn)
		if roundMsg.Type != game.MsgRound {
			t.Fatalf("conn %d: expected ROUND, got %s", i, roundMsg.Type)
		}
		roundParts := strings.SplitN(roundMsg.Payload, "|", 2)
		if len(roundParts) < 2 {
			t.Fatalf("malformed ROUND payload: %q", roundMsg.Payload)
		}
		if i == 0 {
			speakers = strings.Split(roundParts[1], ",")
		}

		turnMsg := readMsgFromConn(t, conn)
		if turnMsg.Type != game.MsgTurn {
			t.Fatalf("conn %d: expected TURN, got %s", i, turnMsg.Type)
		}
	}

	return conns, speakers, cleanup
}

// doDescription drives the description phase: each speaker describes in order.
func doDescription(t *testing.T, conns []net.Conn, speakers []string) {
	t.Helper()
	for si, speaker := range speakers {
		speakerConn := findConnByName(conns, speaker)
		fmt.Fprintf(speakerConn, "DESC|desc from %s\n", speaker)

		// All players receive DESC broadcast.
		for _, conn := range conns {
			msg := readMsgFromConn(t, conn)
			if msg.Type != game.MsgDesc {
				t.Fatalf("expected DESC, got %s: %s", msg.Type, msg.Payload)
			}
		}

		// After DESC, TURN for next speaker (unless last).
		if si < len(speakers)-1 {
			for _, conn := range conns {
				msg := readMsgFromConn(t, conn)
				if msg.Type != game.MsgTurn {
					t.Fatalf("expected TURN, got %s: %s", msg.Type, msg.Payload)
				}
			}
		}
	}
}

// doVotingForElimination drives the voting phase targeting a specific player for elimination.
// target is the player to be eliminated (gets majority votes).
// Returns the RESULT message from conns[0].
func doVotingForElimination(t *testing.T, conns []net.Conn, voters []string, target string) game.Message {
	t.Helper()

	// Read VOTE + TURN from all connections.
	for _, conn := range conns {
		readMsgFromConn(t, conn) // VOTE
		readMsgFromConn(t, conn) // TURN
	}

	for vi, voter := range voters {
		voterConn := findConnByName(conns, voter)
		if voter == target {
			// Target votes for someone else (can't vote for self).
			other := voters[0]
			if other == target {
				other = voters[1]
			}
			fmt.Fprintf(voterConn, "VOTE|%s\n", other)
		} else {
			fmt.Fprintf(voterConn, "VOTE|%s\n", target)
		}

		if vi < len(voters)-1 {
			for _, conn := range conns {
				readMsgFromConn(t, conn) // TURN for next voter
			}
		}
	}

	// Read RESULT from all connections.
	resultMsg := readMsgFromConn(t, conns[0])
	if resultMsg.Type != game.MsgResult {
		t.Fatalf("expected RESULT, got %s: %s", resultMsg.Type, resultMsg.Payload)
	}
	for i := 1; i < len(conns); i++ {
		readMsgFromConn(t, conns[i])
	}
	return resultMsg
}

func TestGameLoop_WinBroadcastOnCivilianWin(t *testing.T) {
	conns, speakers, cleanup := setupGameLoopTest(t, 4)
	defer cleanup()

	// Round 1: all describe.
	doDescription(t, conns, speakers)

	// Round 1: vote out the undercover. With 1U, eliminating the U should end the game.
	// We don't know which player is the undercover, but we can vote out speakers[0]
	// and check if the game ends. If it doesn't end, we continue.
	// For a deterministic test, let's vote out speakers[0].
	eliminated := speakers[0]
	doVotingForElimination(t, conns, speakers, eliminated)

	// After voting, all connections should receive either:
	// - WIN message (if undercover was eliminated)
	// - ROUND message (if game continues to next round)
	msg := readMsgFromConn(t, conns[0])

	if msg.Type == game.MsgWin {
		// Undercover was eliminated — civilians win.
		// Verify WIN payload format: "winner|playerStates|civilianWord|undercoverWord"
		parts := strings.SplitN(msg.Payload, "|", 4)
		if len(parts) < 4 {
			t.Fatalf("WIN payload should have 4 parts, got %d: %q", len(parts), msg.Payload)
		}
		if parts[0] != "Civilian" {
			t.Errorf("winner should be Civilian, got %q", parts[0])
		}
		if parts[2] != "苹果" {
			t.Errorf("civilian word should be 苹果, got %q", parts[2])
		}
		if parts[3] != "香蕉" {
			t.Errorf("undercover word should be 香蕉, got %q", parts[3])
		}
		// Verify player states contain all 4 players.
		states := strings.Split(parts[1], ",")
		if len(states) != 4 {
			t.Errorf("expected 4 player states, got %d: %q", len(states), parts[1])
		}
		// Eliminated player should have alive=0.
		for _, state := range states {
			stateParts := strings.Split(state, ":")
			if len(stateParts) != 3 {
				t.Errorf("invalid player state format: %q", state)
				continue
			}
			name, role, alive := stateParts[0], stateParts[1], stateParts[2]
			if name == eliminated && alive != "0" {
				t.Errorf("eliminated player %s should have alive=0, got %q", eliminated, alive)
			}
			if role != "Civilian" && role != "Undercover" && role != "Blank" {
				t.Errorf("unexpected role in state: %q", role)
			}
		}

		// Verify remaining connections also receive WIN.
		for i := 1; i < len(conns); i++ {
			winMsg := readMsgFromConn(t, conns[i])
			if winMsg.Type != game.MsgWin {
				t.Errorf("conn %d: expected WIN, got %s: %s", i, winMsg.Type, winMsg.Payload)
			}
		}
	} else if msg.Type == game.MsgRound {
		// Game continues — we eliminated a civilian.
		// Verify it's round 2.
		roundParts := strings.SplitN(msg.Payload, "|", 2)
		if len(roundParts) < 2 {
			t.Fatalf("malformed ROUND payload: %q", msg.Payload)
		}
		if roundParts[0] != "2" {
			t.Errorf("expected round 2, got %q", roundParts[0])
		}

		// Verify remaining connections also get ROUND.
		for i := 1; i < len(conns); i++ {
			roundMsg := readMsgFromConn(t, conns[i])
			if roundMsg.Type != game.MsgRound {
				t.Errorf("conn %d: expected ROUND, got %s", i, roundMsg.Type)
			}
		}
	} else {
		t.Fatalf("expected WIN or ROUND after voting, got %s: %s", msg.Type, msg.Payload)
	}
}

func TestGameLoop_GameContinuesToNextRound(t *testing.T) {
	conns, speakers, cleanup := setupGameLoopTest(t, 5)
	defer cleanup()

	// Round 1: all describe.
	doDescription(t, conns, speakers)

	// Round 1: vote out speakers[0]. With 5 players (1U), game should continue
	// unless we happened to eliminate the undercover.
	doVotingForElimination(t, conns, speakers, speakers[0])

	// Read the next message — could be WIN or ROUND.
	msg := readMsgFromConn(t, conns[0])

	if msg.Type == game.MsgWin {
		// We happened to eliminate the undercover. Verify WIN and return.
		if !strings.Contains(msg.Payload, "Civilian") {
			t.Errorf("WIN should show Civilian as winner, got %q", msg.Payload)
		}
		return
	}

	// Game continues to round 2.
	if msg.Type != game.MsgRound {
		t.Fatalf("expected ROUND for round 2, got %s: %s", msg.Type, msg.Payload)
	}
	roundParts := strings.SplitN(msg.Payload, "|", 2)
	if roundParts[0] != "2" {
		t.Errorf("expected round 2, got %q", roundParts[0])
	}

	// Consume TURN from all connections.
	for i, conn := range conns {
		readMsgFromConn(t, conn) // TURN
		if i == 0 {
			// Also consume ROUND+TURN from conns[0] (already read ROUND above).
		}
	}
	// Consume ROUND+TURN from remaining connections.
	for i := 1; i < len(conns); i++ {
		readMsgFromConn(t, conns[i]) // ROUND
		readMsgFromConn(t, conns[i]) // TURN
	}

	// Round 2: speakers are the 4 alive players.
	// Read new speaker order from ROUND message (already consumed above for conns[0]).
	newSpeakers := strings.Split(roundParts[1], ",")

	// Round 2: all describe.
	doDescription(t, conns, newSpeakers)

	// Round 2: vote out the first speaker.
	doVotingForElimination(t, conns, newSpeakers, newSpeakers[0])

	// After round 2, check if game ends or continues.
	msg = readMsgFromConn(t, conns[0])
	if msg.Type == game.MsgWin {
		// Game ended. Verify WIN payload.
		parts := strings.SplitN(msg.Payload, "|", 4)
		if len(parts) < 4 {
			t.Fatalf("WIN payload should have 4 parts, got %d", len(parts))
		}
		// Winner could be Civilian or Undercover depending on who was eliminated.
		if parts[0] != "Civilian" && parts[0] != "Undercover" {
			t.Errorf("unexpected winner: %q", parts[0])
		}
	} else if msg.Type == game.MsgRound {
		// Game continues to round 3. Verify round number.
		roundParts := strings.SplitN(msg.Payload, "|", 2)
		if roundParts[0] != "3" {
			t.Errorf("expected round 3, got %q", roundParts[0])
		}
	} else {
		t.Fatalf("expected WIN or ROUND after round 2, got %s: %s", msg.Type, msg.Payload)
	}
}

func TestBuildWinPayload(t *testing.T) {
	players := []*game.Player{
		{Name: "Alice", Role: game.Civilian, Alive: true},
		{Name: "Bob", Role: game.Undercover, Alive: false},
		{Name: "Charlie", Role: game.Civilian, Alive: true},
		{Name: "Dave", Role: game.Blank, Alive: true},
	}

	payload := buildWinPayload(game.Civilian, players, "苹果", "香蕉")

	parts := strings.SplitN(payload, "|", 4)
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(parts))
	}

	if parts[0] != "Civilian" {
		t.Errorf("winner = %q, want Civilian", parts[0])
	}

	states := strings.Split(parts[1], ",")
	if len(states) != 4 {
		t.Fatalf("expected 4 player states, got %d", len(states))
	}

	expectedStates := map[string]string{
		"Alice":   "Civilian:1",
		"Bob":     "Undercover:0",
		"Charlie": "Civilian:1",
		"Dave":    "Blank:1",
	}
	for _, state := range states {
		stateParts := strings.Split(state, ":")
		if len(stateParts) != 3 {
			t.Errorf("invalid state format: %q", state)
			continue
		}
		name := stateParts[0]
		roleAlive := stateParts[1] + ":" + stateParts[2]
		if expected, ok := expectedStates[name]; ok {
			if roleAlive != expected {
				t.Errorf("player %s: got %s, want %s", name, roleAlive, expected)
			}
		} else {
			t.Errorf("unexpected player %q", name)
		}
	}

	if parts[2] != "苹果" {
		t.Errorf("civilian word = %q, want 苹果", parts[2])
	}
	if parts[3] != "香蕉" {
		t.Errorf("undercover word = %q, want 香蕉", parts[3])
	}
}

