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
	if !strings.Contains(out.String(), "room code cannot be empty") {
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
	if !strings.Contains(out.String(), "invalid room code") {
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
	if !strings.Contains(out.String(), "connection failed") {
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
	if !strings.Contains(output, "joined as testPlayer") {
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
	if !strings.Contains(out.String(), "name cannot be empty") {
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
	if !strings.Contains(output, "joined as testPlayer") {
		t.Errorf("expected joined message, got: %s", output)
	}
	if !strings.Contains(output, "assigned token: 苹果 [平民]") {
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
	if !strings.Contains(output, "assigned token: 香蕉 [卧底]") {
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
	if !strings.Contains(output, "assigned token: [白板] — 你是白板") {
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
	if !strings.Contains(output, "assigned token: 苹果 [平民]") {
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
