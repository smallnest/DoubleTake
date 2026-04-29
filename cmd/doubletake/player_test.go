package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/smallnest/doubletake/game"
)

// startTestPlayerServer creates a TCP listener on a random port and returns
// the listener, address string, and the corresponding room code.
func startTestPlayerServer(t *testing.T) (net.Listener, string, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	roomCode := game.EncodeRoomCode(addr)
	if roomCode == "" {
		t.Fatal("failed to encode room code")
	}
	t.Logf("test server on %s, room code: %s", addr, roomCode)
	return ln, addr, roomCode
}

func TestRunPlayer_EmptyStdin(t *testing.T) {
	out := &bytes.Buffer{}
	exitCode := RunPlayer(out, strings.NewReader(""), false)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunPlayer_EmptyRoomCode(t *testing.T) {
	out := &bytes.Buffer{}
	input := "\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(out.String(), "房间码不能为空") {
		t.Errorf("expected room code error, got: %s", out.String())
	}
}

func TestRunPlayer_InvalidRoomCode(t *testing.T) {
	out := &bytes.Buffer{}
	input := "!!!\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(out.String(), "无效房间码") {
		t.Errorf("expected invalid room code error, got: %s", out.String())
	}
}

func TestRunPlayer_ConnectionRefused(t *testing.T) {
	out := &bytes.Buffer{}
	// Pick a high port unlikely to be in use
	roomCode := game.EncodeRoomCode("127.0.0.1:19999")
	input := roomCode + "\nplayer1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(out.String(), "连接失败") {
		t.Errorf("expected connection failed error, got: %s", out.String())
	}
}

func TestRunPlayer_SuccessfulJoin(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
		msg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		// Send JOIN confirmation, then let the connection close
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: msg.Payload}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "已加入为 testPlayer") {
		t.Errorf("expected joined message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ServerErrorResponse(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
		// Reject the join
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgError, Payload: "name taken"}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "name taken") {
		t.Errorf("expected error message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_EmptyPlayerName(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	// Accept but don't respond — player will fail before sending JOIN
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\n\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(out.String(), "玩家名不能为空") {
		t.Errorf("expected name error, got: %s", out.String())
	}
	<-serverDone
}

func TestRunPlayer_StealthMode(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\n"
	// Only room code, no name — will fail at name prompt
	exitCode := RunPlayer(out, strings.NewReader(input), true)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	output := out.String()
	if strings.Contains(output, "[INFO]") {
		t.Errorf("stealth mode should not contain [INFO], got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_NonStealthMode(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("non-stealth mode should contain [INFO], got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesRole_Civilian(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "testPlayer"}))
		
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Civilian|苹果"}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "已加入为 testPlayer") {
		t.Errorf("expected joined message, got: %s", output)
	}
	if !strings.Contains(output, "你的身份: 苹果 [平民]") {
		t.Errorf("expected disguised civilian role message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesRole_Undercover(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "testPlayer"}))
		
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Undercover|香蕉"}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "你的身份: 香蕉 [卧底]") {
		t.Errorf("expected disguised undercover role message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesRole_Blank(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "testPlayer"}))
		
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Blank|你是白板"}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "你的身份: [白板] — 你是白板") {
		t.Errorf("expected disguised blank role message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesRole_Stealth(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "testPlayer"}))
		
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Civilian|苹果"}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), true)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	// Stealth mode should not contain [DATA] tag
	if strings.Contains(output, "[DATA]") {
		t.Errorf("stealth mode should not contain [DATA], got: %s", output)
	}
	// But should still contain the assigned token info
	if !strings.Contains(output, "你的身份: 苹果 [平民]") {
		t.Errorf("expected token display in stealth mode, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesMultipleMessages(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}
		// Send JOIN confirmation, then a READY broadcast, then a ROLE message, then close
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "testPlayer"}))

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgReady, Payload: ""}))

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Civilian|苹果"}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "已加入为 testPlayer") {
		t.Errorf("expected joined message, got: %s", output)
	}
	if !strings.Contains(output, "你的身份: 苹果 [平民]") {
		t.Errorf("expected ROLE message, got: %s", output)
	}
	if !strings.Contains(output, "READY") {
		t.Errorf("expected READY message, got: %s", output)
	}
	<-serverDone
}

// TestRunPlayer_DescPhase_OtherPlayerTurn tests that the player correctly
// displays ROUND, TURN (other player), and DESC broadcast messages.
func TestRunPlayer_DescPhase_OtherPlayerTurn(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "Alice"}))
		// ROUND message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRound, Payload: "1|Bob,Alice,Charlie"}))
		// TURN for Bob (not Alice)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "Bob"}))
		// DESC broadcast from Bob
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgDesc, Payload: "Bob|苹果是红色的"}))
		// TURN for Alice (this is us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "Alice"}))
		// Read the DESC from Alice
		if scanner.Scan() {
			// This is the DESC message from the player; just consume it
		}
	}()

	out := &bytes.Buffer{}
	// Input: room code, name, then description for Alice's turn
	input := roomCode + "\nAlice\n苹果很好吃\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify ROUND display
	if !strings.Contains(output, "轮次 1，发言顺序: Bob → Alice → Charlie") {
		t.Errorf("expected formatted ROUND display, got: %s", output)
	}

	// Verify waiting message for other player's turn
	if !strings.Contains(output, "等待 Bob 描述...") {
		t.Errorf("expected waiting message for Bob, got: %s", output)
	}

	// Verify DESC display from other player
	if !strings.Contains(output, "Bob: 苹果是红色的") {
		t.Errorf("expected formatted DESC display from Bob, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_DescPhase_OwnTurn tests that the player correctly prompts
// for input when receiving TURN for itself and sends the description.
func TestRunPlayer_DescPhase_OwnTurn(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "P0"}))
		// ROUND message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRound, Payload: "1|P0,P1"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Read the DESC message from P0
		if !scanner.Scan() {
			return
		}
		msg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if msg.Type != game.MsgDesc {
			t.Errorf("expected DESC message, got %s", msg.Type)
			return
		}
		if msg.Payload != "苹果很好吃" {
			t.Errorf("expected DESC payload '苹果很好吃', got '%s'", msg.Payload)
			return
		}
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nP0\n苹果很好吃\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify prompt displayed
	if !strings.Contains(output, "请输入描述:") {
		t.Errorf("expected description prompt, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_DescPhase_EmptyDescRetry tests that the player re-prompts
// when the user enters an empty description (client-side check).
func TestRunPlayer_DescPhase_EmptyDescRetry(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "P0"}))
		// ROUND message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRound, Payload: "1|P0"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Client-side empty check blocks sending empty, so the first non-empty
		// desc will arrive here.
		if !scanner.Scan() {
			return
		}
		msg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if msg.Payload != "valid description" {
			t.Errorf("expected 'valid description', got '%s'", msg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	// Input: room code, name, empty line (caught client-side), valid desc
	input := roomCode + "\nP0\n\nvalid description\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify client-side empty warning
	if !strings.Contains(output, "描述不能为空") {
		t.Errorf("expected empty description warning, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_DescPhase_ServerErrorRetry tests that the player re-prompts
// when the server rejects a description with ERROR (server-side rejection).
func TestRunPlayer_DescPhase_ServerErrorRetry(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "P0"}))
		// ROUND message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRound, Payload: "1|P0"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Read first DESC
		if !scanner.Scan() {
			return
		}
		// Send ERROR to reject (e.g. "还没轮到你发言")
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgError, Payload: "描述不能为空，请重新输入"}))
		// Read second DESC
		if !scanner.Scan() {
			return
		}
		msg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if msg.Payload != "valid description" {
			t.Errorf("expected 'valid description', got '%s'", msg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	// Input: room code, name, first desc (rejected by server), valid desc
	input := roomCode + "\nP0\nfirst try\nvalid description\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify error message displayed
	if !strings.Contains(output, "描述不能为空，请重新输入") {
		t.Errorf("expected server error message, got: %s", output)
	}
	// Verify re-prompt
	if strings.Count(output, "请输入描述:") < 2 {
		t.Errorf("expected at least 2 prompts, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_DescPhase_ErrorFatal tests that non-desc-phase ERROR
// messages cause the player to exit.
func TestRunPlayer_DescPhase_ErrorFatal(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// Send ERROR immediately (not during desc phase)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgError, Payload: "game is full"}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nPlayer1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "game is full") {
		t.Errorf("expected error message, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_VotePhase_OtherPlayerTurn tests that the player correctly
// displays VOTE, TURN (other player) messages during the voting phase.
func TestRunPlayer_VotePhase_OtherPlayerTurn(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "Alice"}))
		// VOTE message: round 1, alive players
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgVote, Payload: "1|Bob,Alice"}))
		// TURN for Bob (not Alice)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "Bob"}))
		// TURN for Alice (us) — need to read her vote
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "Alice"}))
		// Read the VOTE message from Alice
		if scanner.Scan() {
			// consume the vote
		}
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nAlice\nBob\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify VOTE display
	if !strings.Contains(output, "投票环节 轮次 1") {
		t.Errorf("expected vote round display, got: %s", output)
	}
	if !strings.Contains(output, "可投票: Bob → Alice") {
		t.Errorf("expected alive player list, got: %s", output)
	}

	// Verify waiting message for other player's turn
	if !strings.Contains(output, "等待 Bob 投票...") {
		t.Errorf("expected waiting message for Bob voting, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_VotePhase_OwnTurn tests that the player correctly prompts
// for vote input when receiving TURN for itself and sends VOTE|targetName.
func TestRunPlayer_VotePhase_OwnTurn(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "P0"}))
		// VOTE message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgVote, Payload: "1|P0,P1"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Read the VOTE message from P0
		if !scanner.Scan() {
			return
		}
		msg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if msg.Type != game.MsgVote {
			t.Errorf("expected VOTE message, got %s", msg.Type)
			return
		}
		if msg.Payload != "P1" {
			t.Errorf("expected VOTE payload 'P1', got '%s'", msg.Payload)
			return
		}
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nP0\nP1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify prompt displayed
	if !strings.Contains(output, "请输入投票目标:") {
		t.Errorf("expected vote prompt, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_VotePhase_EmptyTargetRetry tests that the player re-prompts
// when the user enters an empty vote target (client-side check).
func TestRunPlayer_VotePhase_EmptyTargetRetry(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "P0"}))
		// VOTE message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgVote, Payload: "1|P0,P1"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Client-side empty check blocks sending empty, so the first non-empty
		// vote will arrive here.
		if !scanner.Scan() {
			return
		}
		msg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if msg.Payload != "P1" {
			t.Errorf("expected 'P1', got '%s'", msg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	// Input: room code, name, empty line (caught client-side), valid target
	input := roomCode + "\nP0\n\nP1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify client-side empty warning
	if !strings.Contains(output, "投票目标不能为空") {
		t.Errorf("expected empty vote target warning, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_VotePhase_ServerErrorRetry tests that the player re-prompts
// when the server rejects a vote with ERROR (server-side rejection).
func TestRunPlayer_VotePhase_ServerErrorRetry(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "P0"}))
		// VOTE message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgVote, Payload: "1|P0,P1"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Read first VOTE (will be rejected)
		if !scanner.Scan() {
			return
		}
		// Send ERROR to reject (e.g. "cannot vote for yourself")
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgError, Payload: "cannot vote for yourself"}))
		// Read second VOTE
		if !scanner.Scan() {
			return
		}
		msg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if msg.Payload != "P1" {
			t.Errorf("expected 'P1', got '%s'", msg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	// Input: room code, name, self vote (rejected by server), valid target
	input := roomCode + "\nP0\nP0\nP1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify error message displayed
	if !strings.Contains(output, "cannot vote for yourself") {
		t.Errorf("expected server error message, got: %s", output)
	}
	// Verify re-prompt
	if strings.Count(output, "请输入投票目标:") < 2 {
		t.Errorf("expected at least 2 vote prompts, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_VotePhase_ResultDisplay tests that the player correctly
// displays the RESULT message with vote tallies.
func TestRunPlayer_VotePhase_ResultDisplay(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "Alice"}))
		// VOTE message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgVote, Payload: "1|Bob,Alice"}))
		// TURN for Alice (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "Alice"}))
		// Read the VOTE from Alice
		if scanner.Scan() {
			// consume the vote
		}
		// Send RESULT
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgResult, Payload: "Bob:2,Alice:1"}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nAlice\nBob\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify RESULT display
	if !strings.Contains(output, "投票结果:") {
		t.Errorf("expected result header, got: %s", output)
	}
	if !strings.Contains(output, "Bob: 2 票") {
		t.Errorf("expected Bob vote count, got: %s", output)
	}
	if !strings.Contains(output, "Alice: 1 票") {
		t.Errorf("expected Alice vote count, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_VotePhase_DescToVoteTransition tests that the player correctly
// transitions from description phase to voting phase.
func TestRunPlayer_VotePhase_DescToVoteTransition(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "P0"}))
		// Description phase
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRound, Payload: "1|P0"}))
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Read DESC from P0
		if !scanner.Scan() {
			return
		}
		// Broadcast DESC
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgDesc, Payload: "P0|something"}))
		// Voting phase starts
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgVote, Payload: "1|P0,P1"}))
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Read VOTE from P0
		if !scanner.Scan() {
			return
		}
		msg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if msg.Type != game.MsgVote || msg.Payload != "P1" {
			t.Errorf("expected VOTE P1, got %s %s", msg.Type, msg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nP0\nsomething\nP1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Verify desc phase was shown
	if !strings.Contains(output, "请输入描述:") {
		t.Errorf("expected desc prompt, got: %s", output)
	}
	// Verify vote phase started
	if !strings.Contains(output, "投票环节 轮次 1") {
		t.Errorf("expected vote round display, got: %s", output)
	}
	if !strings.Contains(output, "请输入投票目标:") {
		t.Errorf("expected vote prompt, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_ReceivesWin_CivilianWins tests that the player correctly
// parses and displays a WIN message when civilians win, and exits with code 0.
func TestRunPlayer_ReceivesWin_CivilianWins(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "Alice"}))
		// WIN message: civilians win
		winPayload := "Civilian|Alice:Civilian:1,Bob:Undercover:0,Charlie:Civilian:1|苹果|香蕉"
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgWin, Payload: winPayload}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nAlice\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	if !strings.Contains(output, "游戏结束 — 平民 胜利") {
		t.Errorf("expected civilian win message, got: %s", output)
	}
	if !strings.Contains(output, "Alice [平民] 存活") {
		t.Errorf("expected Alice alive, got: %s", output)
	}
	if !strings.Contains(output, "Bob [卧底] 已淘汰") {
		t.Errorf("expected Bob eliminated, got: %s", output)
	}
	if !strings.Contains(output, "平民词语: 苹果") {
		t.Errorf("expected civilian word, got: %s", output)
	}
	if !strings.Contains(output, "卧底词语: 香蕉") {
		t.Errorf("expected undercover word, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_ReceivesWin_UndercoverWins tests that the player correctly
// displays a WIN message when undercover wins.
func TestRunPlayer_ReceivesWin_UndercoverWins(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "Bob"}))
		// WIN message: undercover wins
		winPayload := "Undercover|Alice:Civilian:0,Bob:Undercover:1,Charlie:Civilian:0|苹果|香蕉"
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgWin, Payload: winPayload}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nBob\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	if !strings.Contains(output, "游戏结束 — 卧底 胜利") {
		t.Errorf("expected undercover win message, got: %s", output)
	}
	if !strings.Contains(output, "Bob [卧底] 存活") {
		t.Errorf("expected Bob alive, got: %s", output)
	}
	if !strings.Contains(output, "Alice [平民] 已淘汰") {
		t.Errorf("expected Alice eliminated, got: %s", output)
	}

	<-serverDone
}

// TestRunPlayer_ReceivesWin_Stealth tests WIN display in stealth mode.
func TestRunPlayer_ReceivesWin_Stealth(t *testing.T) {
	ln, _, roomCode := startTestPlayerServer(t)
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			return
		}

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "Alice"}))
		winPayload := "Civilian|Alice:Civilian:1,Bob:Undercover:0|苹果|香蕉"
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgWin, Payload: winPayload}))
	}()

	out := &bytes.Buffer{}
	input := roomCode + "\nAlice\n"
	exitCode := RunPlayer(out, strings.NewReader(input), true)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()

	// Stealth mode should not contain [DATA] tag
	if strings.Contains(output, "[DATA]") {
		t.Errorf("stealth mode should not contain [DATA], got: %s", output)
	}
	// But should still contain the result info
	if !strings.Contains(output, "游戏结束 — 平民 胜利") {
		t.Errorf("expected win message in stealth mode, got: %s", output)
	}

	<-serverDone
}
