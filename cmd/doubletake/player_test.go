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

// testPassword is a constant used in player tests for the room password.
const testPassword = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

// startTestPlayerServer creates a TCP listener on a random port and returns
// the listener and address string.
func startTestPlayerServer(t *testing.T) (net.Listener, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	t.Logf("test server on %s", addr)
	return ln, addr
}

func TestRunPlayer_EmptyStdin(t *testing.T) {
	out := &bytes.Buffer{}
	addr := "127.0.0.1:19999"
	exitCode := RunPlayer(out, strings.NewReader(""), false, addr)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunPlayer_EmptyPassword(t *testing.T) {
	out := &bytes.Buffer{}
	addr := "127.0.0.1:19999"
	input := "\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(out.String(), "password cannot be empty") {
		t.Errorf("expected password error, got: %s", out.String())
	}
}

func TestRunPlayer_ConnectionRefused(t *testing.T) {
	out := &bytes.Buffer{}
	// Pick a high port unlikely to be in use
	addr := "127.0.0.1:19999"
	input := testPassword + "\nplayer1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(out.String(), "connection failed") {
		t.Errorf("expected connection failed error, got: %s", out.String())
	}
}

func TestRunPlayer_SuccessfulJoin(t *testing.T) {
	ln, addr := startTestPlayerServer(t)
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
		// Extract name from "hash|playerName" and echo back just the name
		parts := strings.SplitN(msg.Payload, "|", 2)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "joined as testPlayer") {
		t.Errorf("expected joined message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ServerErrorResponse(t *testing.T) {
	ln, addr := startTestPlayerServer(t)
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
		// Consume the JOIN message, then reject
		_, _ = game.Decode(scanner.Text())
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgError, Payload: "name taken"}))
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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
	input := testPassword + "\n\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(out.String(), "name cannot be empty") {
		t.Errorf("expected name error, got: %s", out.String())
	}
	<-serverDone
}

func TestRunPlayer_StealthMode(t *testing.T) {
	ln, addr := startTestPlayerServer(t)
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
	input := testPassword + "\n"
	// Only password, no name — will fail at name prompt
	exitCode := RunPlayer(out, strings.NewReader(input), true, addr)

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
	ln, addr := startTestPlayerServer(t)
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
	input := testPassword + "\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Civilian|苹果"}))
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "joined as testPlayer") {
		t.Errorf("expected joined message, got: %s", output)
	}
	if !strings.Contains(output, "assigned token: 苹果 [平民]") {
		t.Errorf("expected disguised civilian role message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesRole_Undercover(t *testing.T) {
	ln, addr := startTestPlayerServer(t)
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
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Undercover|香蕉"}))
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "assigned token: 香蕉 [卧底]") {
		t.Errorf("expected disguised undercover role message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesRole_Blank(t *testing.T) {
	ln, addr := startTestPlayerServer(t)
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
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Blank|你是白板"}))
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "assigned token: [白板] — 你是白板") {
		t.Errorf("expected disguised blank role message, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesRole_Stealth(t *testing.T) {
	ln, addr := startTestPlayerServer(t)
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
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Civilian|苹果"}))
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), true, addr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	// Stealth mode should not contain [DATA] tag
	if strings.Contains(output, "[DATA]") {
		t.Errorf("stealth mode should not contain [DATA], got: %s", output)
	}
	// But should still contain the assigned token info
	if !strings.Contains(output, "assigned token: 苹果 [平民]") {
		t.Errorf("expected token display in stealth mode, got: %s", output)
	}
	<-serverDone
}

func TestRunPlayer_ReceivesMultipleMessages(t *testing.T) {
	ln, addr := startTestPlayerServer(t)
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
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// Send JOIN confirmation, then a READY broadcast, then a ROLE message, then close
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgReady, Payload: ""}))

		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRole, Payload: "Civilian|苹果"}))
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\ntestPlayer\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := out.String()
	if !strings.Contains(output, "joined as testPlayer") {
		t.Errorf("expected joined message, got: %s", output)
	}
	if !strings.Contains(output, "assigned token: 苹果 [平民]") {
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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
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
	// Input: password, name, then description for Alice's turn
	input := testPassword + "\nAlice\n苹果很好吃\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
		// ROUND message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRound, Payload: "1|P0,P1"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Read the DESC message from P0
		if !scanner.Scan() {
			return
		}
		descMsg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if descMsg.Type != game.MsgDesc {
			t.Errorf("expected DESC message, got %s", descMsg.Type)
			return
		}
		if descMsg.Payload != "苹果很好吃" {
			t.Errorf("expected DESC payload '苹果很好吃', got '%s'", descMsg.Payload)
			return
		}
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\nP0\n苹果很好吃\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
		// ROUND message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgRound, Payload: "1|P0"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Client-side empty check blocks sending empty, so the first non-empty
		// desc will arrive here.
		if !scanner.Scan() {
			return
		}
		descMsg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if descMsg.Payload != "valid description" {
			t.Errorf("expected 'valid description', got '%s'", descMsg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	// Input: password, name, empty line (caught client-side), valid desc
	input := testPassword + "\nP0\n\nvalid description\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
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
		descMsg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if descMsg.Payload != "valid description" {
			t.Errorf("expected 'valid description', got '%s'", descMsg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	// Input: password, name, first desc (rejected by server), valid desc
	input := testPassword + "\nP0\nfirst try\nvalid description\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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
		_, _ = game.Decode(scanner.Text())
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgError, Payload: "game is full"}))
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\nPlayer1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
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
	input := testPassword + "\nAlice\nBob\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
		// VOTE message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgVote, Payload: "1|P0,P1"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Read the VOTE message from P0
		if !scanner.Scan() {
			return
		}
		voteMsg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if voteMsg.Type != game.MsgVote {
			t.Errorf("expected VOTE message, got %s", voteMsg.Type)
			return
		}
		if voteMsg.Payload != "P1" {
			t.Errorf("expected VOTE payload 'P1', got '%s'", voteMsg.Payload)
			return
		}
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\nP0\nP1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
		// VOTE message
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgVote, Payload: "1|P0,P1"}))
		// TURN for P0 (us)
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgTurn, Payload: "P0"}))
		// Client-side empty check blocks sending empty, so the first non-empty
		// vote will arrive here.
		if !scanner.Scan() {
			return
		}
		voteMsg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if voteMsg.Payload != "P1" {
			t.Errorf("expected 'P1', got '%s'", voteMsg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	// Input: password, name, empty line (caught client-side), valid target
	input := testPassword + "\nP0\n\nP1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
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
		voteMsg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if voteMsg.Payload != "P1" {
			t.Errorf("expected 'P1', got '%s'", voteMsg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	// Input: password, name, self vote (rejected by server), valid target
	input := testPassword + "\nP0\nP0\nP1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
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
	input := testPassword + "\nAlice\nBob\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
	ln, addr := startTestPlayerServer(t)
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

		// Consume JOIN, extract name
		msg, _ := game.Decode(scanner.Text())
		parts := strings.SplitN(msg.Payload, "|", 2)
		// JOIN confirm
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: parts[1]}))
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
		voteMsg, err := game.Decode(scanner.Text())
		if err != nil {
			return
		}
		if voteMsg.Type != game.MsgVote || voteMsg.Payload != "P1" {
			t.Errorf("expected VOTE P1, got %s %s", voteMsg.Type, voteMsg.Payload)
		}
	}()

	out := &bytes.Buffer{}
	input := testPassword + "\nP0\nsomething\nP1\n"
	exitCode := RunPlayer(out, strings.NewReader(input), false, addr)

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
