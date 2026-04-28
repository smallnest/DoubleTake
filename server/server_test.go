package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/smallnest/doubletake/game"
)

func startTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	srv := NewServer(port)
	go srv.Start()

	// Wait for server to be ready
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 10*time.Millisecond)
		if err == nil {
			conn.Close()
			return srv, port
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server did not start in time")
	return nil, ""
}

func TestNewServer(t *testing.T) {
	srv := NewServer("9999")
	if srv.port != "9999" {
		t.Errorf("expected port 9999, got %s", srv.port)
	}
	if srv.connections == nil {
		t.Error("connections map should be initialized")
	}
	if len(srv.connections) != 0 {
		t.Error("connections map should be empty")
	}
}

func TestStartAndAccept(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Give server time to register connection
	time.Sleep(50 * time.Millisecond)

	srv.mu.Lock()
	count := len(srv.connections)
	srv.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 connection, got %d", count)
	}
}

func TestStop(t *testing.T) {
	srv, port := startTestServer(t)

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	srv.Stop()

	// Connection should be closed by server
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err == nil {
		t.Error("expected connection to be closed after Stop()")
	}
}

func TestStopIdempotent(t *testing.T) {
	srv, _ := startTestServer(t)

	// Calling Stop() twice should not panic
	srv.Stop()
	srv.Stop()
}

func TestHandleConnMessages(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send a valid message
	fmt.Fprintf(conn, "JOIN|player1\n")

	// Send an invalid message (should not crash server)
	fmt.Fprintf(conn, "invalid\n")

	// Send another valid message
	fmt.Fprintf(conn, "READY|\n")

	// Give server time to process
	time.Sleep(50 * time.Millisecond)

	// Server should still be running and connection alive
	srv.mu.Lock()
	count := len(srv.connections)
	srv.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 connection after messages, got %d", count)
	}
}

func TestClientDisconnect(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	srv.mu.Lock()
	before := len(srv.connections)
	srv.mu.Unlock()

	if before != 1 {
		t.Fatalf("expected 1 connection before disconnect, got %d", before)
	}

	// Close client connection — server should handle gracefully
	conn.Close()

	// Wait for server to detect disconnect
	time.Sleep(100 * time.Millisecond)

	srv.mu.Lock()
	after := len(srv.connections)
	srv.mu.Unlock()

	if after != 0 {
		t.Errorf("expected 0 connections after disconnect, got %d", after)
	}
}

func TestBroadcast(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	numClients := 3
	conns := make([]net.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		conns[i] = conn
		defer conn.Close()
	}

	time.Sleep(50 * time.Millisecond)

	msg := game.Message{Type: game.MsgStart, Payload: "round1"}
	srv.Broadcast(msg)

	for i, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("client %d failed to read broadcast: %v", i, err)
			continue
		}
		expected := game.Encode(msg)
		if line != expected {
			t.Errorf("client %d: expected %q, got %q", i, expected, line)
		}
	}
}

func TestSend(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Find the server-side connection
	srv.mu.Lock()
	var serverConn net.Conn
	for c := range srv.connections {
		serverConn = c
		break
	}
	srv.mu.Unlock()

	if serverConn == nil {
		t.Fatal("no server-side connection found")
	}

	msg := game.Message{Type: game.MsgRole, Payload: "undercover"}
	srv.Send(serverConn, msg)

	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read sent message: %v", err)
	}

	expected := game.Encode(msg)
	if line != expected {
		t.Errorf("expected %q, got %q", expected, line)
	}
}

func TestMultipleClients(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	numClients := 5
	conns := make([]net.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		conns[i] = conn
		defer conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	srv.mu.Lock()
	count := len(srv.connections)
	srv.mu.Unlock()

	if count != numClients {
		t.Errorf("expected %d connections, got %d", numClients, count)
	}

	// Disconnect half the clients
	for i := 0; i < 3; i++ {
		conns[i].Close()
	}

	time.Sleep(100 * time.Millisecond)

	srv.mu.Lock()
	count = len(srv.connections)
	srv.mu.Unlock()

	if count != numClients-3 {
		t.Errorf("expected %d connections after partial disconnect, got %d", numClients-3, count)
	}
}

func TestInvalidMessageNoPanic(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send various invalid messages
	invalidMessages := []string{
		"\n",
		"NOPIPE\n",
		"||\n",
		"ONLYTYPE\n",
		strings.Repeat("X", 10000) + "\n",
	}

	for _, msg := range invalidMessages {
		fmt.Fprint(conn, msg)
	}

	// Send a valid message after invalid ones to verify server is still alive
	fmt.Fprintf(conn, "JOIN|test\n")

	time.Sleep(50 * time.Millisecond)

	// Verify server still tracks connection
	srv.mu.Lock()
	count := len(srv.connections)
	srv.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 connection after invalid messages, got %d", count)
	}
}
