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

	conns := make([]net.Conn, 4)
	for i, name := range []string{"A", "B", "C", "D"} {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect %s: %v", name, err)
		}
		conns[i] = conn
		defer conn.Close()
		fmt.Fprintf(conn, "JOIN|%s\n", name)
	}

	waitForOutput(t, out, "人已齐，输入 start 开始游戏", 2*time.Second)

	// Consume the JOIN confirmation from each player connection first
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
	waitForOutput(t, out, "游戏开始！", 2*time.Second)

	// Verify players receive READY message
	for i, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(time.Second))
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("player %d did not receive READY: %v", i, err)
			continue
		}
		msg, err := game.Decode(line)
		if err != nil {
			t.Errorf("player %d: decode error: %v", i, err)
			continue
		}
		if msg.Type != game.MsgReady {
			t.Errorf("player %d: expected READY, got %s", i, msg.Type)
		}
	}
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
	waitForOutput(t, out, "游戏开始！", 2*time.Second)
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
