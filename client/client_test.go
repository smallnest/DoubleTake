package client

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/smallnest/doubletake/game"
)

// startTestServer creates a TCP server on a random port and returns the listener and address.
func startTestServer(t *testing.T) (net.Listener, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	return ln, addr
}

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c.messages == nil {
		t.Error("messages channel should be initialized")
	}
	if c.done == nil {
		t.Error("done channel should be initialized")
	}
}

func TestConnectSuccess(t *testing.T) {
	ln, addr := startTestServer(t)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}()

	c := NewClient()
	err := c.Connect(addr)
	if err != nil {
		t.Fatalf("expected successful connection, got: %v", err)
	}
	defer c.Disconnect()
}

func TestConnectFailure(t *testing.T) {
	c := NewClient()
	err := c.Connect("127.0.0.1:1") // port 1 should be unavailable
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if err.Error() == "" {
		t.Fatal("error message should not be empty")
	}
}

func TestSendNotConnected(t *testing.T) {
	c := NewClient()
	err := c.Send(game.Message{Type: game.MsgJoin, Payload: "player1"})
	if err == nil {
		t.Fatal("expected error when sending on disconnected client")
	}
}

func TestSendAndReceive(t *testing.T) {
	ln, addr := startTestServer(t)
	defer ln.Close()

	serverConnCh := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		serverConnCh <- conn
	}()

	c := NewClient()
	err := c.Connect(addr)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// Get the server-side connection
	var serverConn net.Conn
	select {
	case serverConn = <-serverConnCh:
	case <-time.After(time.Second):
		t.Fatal("server did not accept connection")
	}
	defer serverConn.Close()

	// Client sends a message to server
	msg := game.Message{Type: game.MsgJoin, Payload: "player1"}
	if err := c.Send(msg); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	// Server reads the message
	serverConn.SetReadDeadline(time.Now().Add(time.Second))
	reader := bufio.NewReader(serverConn)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("server failed to read: %v", err)
	}
	expected := game.Encode(msg)
	if line != expected {
		t.Errorf("server: expected %q, got %q", expected, line)
	}
}

func TestReceiveFromServer(t *testing.T) {
	ln, addr := startTestServer(t)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Server sends a message
		msg := game.Message{Type: game.MsgRole, Payload: "spy"}
		fmt.Fprint(conn, game.Encode(msg))
	}()

	c := NewClient()
	err := c.Connect(addr)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// Client should receive the message via channel
	select {
	case msg := <-c.Messages():
		if msg.Type != game.MsgRole {
			t.Errorf("expected type %q, got %q", game.MsgRole, msg.Type)
		}
		if msg.Payload != "spy" {
			t.Errorf("expected payload %q, got %q", "spy", msg.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message from server")
	}
}

func TestDisconnect(t *testing.T) {
	ln, addr := startTestServer(t)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}()

	c := NewClient()
	if err := c.Connect(addr); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	// Calling Disconnect multiple times should not panic
	c.Disconnect()
	c.Disconnect()
}

func TestServerDisconnectDetectsEOF(t *testing.T) {
	ln, addr := startTestServer(t)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Server immediately closes connection
		conn.Close()
	}()

	c := NewClient()
	if err := c.Connect(addr); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// Client should detect EOF and close messages channel
	_, ok := <-c.Messages()
	if ok {
		t.Error("expected messages channel to be closed on server disconnect")
	}
}

func TestMultipleMessages(t *testing.T) {
	ln, addr := startTestServer(t)
	defer ln.Close()

	msgs := []game.Message{
		{Type: game.MsgStart, Payload: "round1"},
		{Type: game.MsgWord, Payload: "apple"},
		{Type: game.MsgTurn, Payload: "player1"},
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		for _, msg := range msgs {
			fmt.Fprint(conn, game.Encode(msg))
		}
	}()

	c := NewClient()
	if err := c.Connect(addr); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	for i, expected := range msgs {
		select {
		case msg := <-c.Messages():
			if msg.Type != expected.Type || msg.Payload != expected.Payload {
				t.Errorf("msg %d: expected {%q,%q}, got {%q,%q}", i, expected.Type, expected.Payload, msg.Type, msg.Payload)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for message %d", i)
		}
	}
}

func TestInvalidMessagesSkipped(t *testing.T) {
	ln, addr := startTestServer(t)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Send invalid messages followed by a valid one
		fmt.Fprintf(conn, "invalid\n")
		fmt.Fprintf(conn, "\n")
		fmt.Fprintf(conn, "NOPIPE\n")
		fmt.Fprint(conn, game.Encode(game.Message{Type: game.MsgJoin, Payload: "ok"}))
	}()

	c := NewClient()
	if err := c.Connect(addr); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// Should only receive the valid message
	select {
	case msg := <-c.Messages():
		if msg.Type != game.MsgJoin || msg.Payload != "ok" {
			t.Errorf("expected JOIN|ok, got %s|%s", msg.Type, msg.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for valid message")
	}
}
