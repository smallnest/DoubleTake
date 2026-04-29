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

	srv := NewServer(port, 6)
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
	srv := NewServer("9999", 6)
	if srv.port != "9999" {
		t.Errorf("expected port 9999, got %s", srv.port)
	}
	if srv.totalPlayers != 6 {
		t.Errorf("expected totalPlayers 6, got %d", srv.totalPlayers)
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

func TestSendToPlayer_Success(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Join with a name
	fmt.Fprintf(conn, "JOIN|Alice\n")
	readMsg(t, conn)

	msg := game.Message{Type: game.MsgRole, Payload: "Undercover|香蕉"}
	err = srv.SendToPlayer("Alice", msg)
	if err != nil {
		t.Fatalf("SendToPlayer failed: %v", err)
	}

	received := readMsg(t, conn)
	if received.Type != game.MsgRole {
		t.Errorf("expected ROLE, got %s", received.Type)
	}
	if received.Payload != "Undercover|香蕉" {
		t.Errorf("expected payload %q, got %q", "Undercover|香蕉", received.Payload)
	}
}

func TestSendToPlayer_NotFound(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "JOIN|Alice\n")
	readMsg(t, conn)

	msg := game.Message{Type: game.MsgRole, Payload: "test"}
	err = srv.SendToPlayer("Bob", msg)
	if err == nil {
		t.Fatal("expected error for non-existent player")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestSendToPlayer_OnlyTargetReceives(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn1, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn2.Close()

	fmt.Fprintf(conn1, "JOIN|Alice\n")
	readMsg(t, conn1)

	fmt.Fprintf(conn2, "JOIN|Bob\n")
	readMsg(t, conn2)

	msg := game.Message{Type: game.MsgRole, Payload: "Civilian|苹果"}
	err = srv.SendToPlayer("Alice", msg)
	if err != nil {
		t.Fatalf("SendToPlayer failed: %v", err)
	}

	// Alice should receive it
	received := readMsg(t, conn1)
	if received.Type != game.MsgRole {
		t.Errorf("expected ROLE, got %s", received.Type)
	}

	// Bob should NOT receive anything
	conn2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 128)
	n, err := conn2.Read(buf)
	if err == nil && n > 0 {
		t.Errorf("Bob should not receive the message, got: %q", buf[:n])
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

func readMsg(t *testing.T, conn net.Conn) game.Message {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	reader := bufio.NewReader(conn)
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

func TestJoinSuccess(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "JOIN|Alice\n")

	msg := readMsg(t, conn)
	if msg.Type != game.MsgJoin {
		t.Errorf("expected JOIN response, got %s", msg.Type)
	}
	if msg.Payload != "Alice" {
		t.Errorf("expected payload 'Alice', got %q", msg.Payload)
	}
}

func TestJoinDuplicateName(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	// First client joins as Alice
	conn1, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn1.Close()

	fmt.Fprintf(conn1, "JOIN|Alice\n")
	readMsg(t, conn1) // consume JOIN confirmation

	// Second client tries to join with the same name
	conn2, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn2.Close()

	fmt.Fprintf(conn2, "JOIN|Alice\n")

	msg := readMsg(t, conn2)
	if msg.Type != game.MsgError {
		t.Errorf("expected ERROR response, got %s", msg.Type)
	}
	if msg.Payload != "名字已存在，请换一个" {
		t.Errorf("unexpected error payload: %q", msg.Payload)
	}

	// Second client can retry with a different name
	fmt.Fprintf(conn2, "JOIN|Bob\n")
	msg = readMsg(t, conn2)
	if msg.Type != game.MsgJoin {
		t.Errorf("expected JOIN response after retry, got %s", msg.Type)
	}
	if msg.Payload != "Bob" {
		t.Errorf("expected payload 'Bob', got %q", msg.Payload)
	}
}

func TestJoinEmptyName(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "JOIN|\n")

	msg := readMsg(t, conn)
	if msg.Type != game.MsgError {
		t.Errorf("expected ERROR response, got %s", msg.Type)
	}
	if msg.Payload != "名字不能为空" {
		t.Errorf("unexpected error payload: %q", msg.Payload)
	}
}

func TestJoinDisconnectCleanup(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	fmt.Fprintf(conn, "JOIN|Alice\n")
	readMsg(t, conn)

	conn.Close()
	time.Sleep(100 * time.Millisecond)

	// New client should be able to use the name "Alice" after disconnect
	conn2, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn2.Close()

	fmt.Fprintf(conn2, "JOIN|Alice\n")

	msg := readMsg(t, conn2)
	if msg.Type != game.MsgJoin {
		t.Errorf("expected JOIN response for reused name, got %s", msg.Type)
	}
}

func TestPlayerCount(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	if count := srv.PlayerCount(); count != 0 {
		t.Errorf("expected 0 players initially, got %d", count)
	}

	conn1, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn1.Close()

	fmt.Fprintf(conn1, "JOIN|Alice\n")
	readMsg(t, conn1)

	if count := srv.PlayerCount(); count != 1 {
		t.Errorf("expected 1 player after Alice joins, got %d", count)
	}

	conn2, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn2.Close()

	fmt.Fprintf(conn2, "JOIN|Bob\n")
	readMsg(t, conn2)

	if count := srv.PlayerCount(); count != 2 {
		t.Errorf("expected 2 players after Bob joins, got %d", count)
	}
}

func TestPlayerNames(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn1, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn1.Close()

	fmt.Fprintf(conn1, "JOIN|Alice\n")
	readMsg(t, conn1)

	conn2, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn2.Close()

	fmt.Fprintf(conn2, "JOIN|Bob\n")
	readMsg(t, conn2)

	names := srv.PlayerNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["Alice"] || !nameSet["Bob"] {
		t.Errorf("expected names Alice and Bob, got %v", names)
	}
}

func TestOnPlayerJoinEvent(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "JOIN|Alice\n")
	readMsg(t, conn)

	select {
	case evt := <-srv.OnPlayerJoin:
		if evt.Name != "Alice" {
			t.Errorf("expected name Alice, got %s", evt.Name)
		}
		if evt.Current != 1 {
			t.Errorf("expected current 1, got %d", evt.Current)
		}
		if evt.Capacity != 6 {
			t.Errorf("expected capacity 6, got %d", evt.Capacity)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for PlayerJoinEvent")
	}
}

func TestBroadcastToNamedPlayers(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	// Connect three clients
	conn1, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect client1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect client2: %v", err)
	}
	defer conn2.Close()

	conn3, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect client3: %v", err)
	}
	defer conn3.Close()

	// Only two of them JOIN
	fmt.Fprintf(conn1, "JOIN|Alice\n")
	readMsg(t, conn1)

	fmt.Fprintf(conn2, "JOIN|Bob\n")
	readMsg(t, conn2)

	// conn3 does NOT join — should not receive broadcast
	time.Sleep(50 * time.Millisecond)

	msg := game.Message{Type: game.MsgReady, Payload: ""}
	srv.BroadcastToNamedPlayers(msg)

	// conn1 and conn2 should receive the message
	for i, conn := range []net.Conn{conn1, conn2} {
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("named client %d failed to read broadcast: %v", i, err)
			continue
		}
		expected := game.Encode(msg)
		if line != expected {
			t.Errorf("named client %d: expected %q, got %q", i, expected, line)
		}
	}

	// conn3 should NOT receive the broadcast
	conn3.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 128)
	n, err := conn3.Read(buf)
	if err == nil && n > 0 {
		t.Errorf("unnamed client should not receive broadcast, got: %q", buf[:n])
	}
}

func TestDesc_UnnamedPlayer(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send DESC without joining — should get ERROR.
	fmt.Fprintf(conn, "DESC|some description\n")

	msg := readMsg(t, conn)
	if msg.Type != game.MsgError {
		t.Errorf("expected ERROR for unnamed DESC, got %s", msg.Type)
	}
	if msg.Payload != "请先加入游戏" {
		t.Errorf("unexpected error payload: %q", msg.Payload)
	}
}

func TestDesc_NamedPlayerForwarded(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "JOIN|Alice\n")
	readMsg(t, conn) // consume JOIN confirmation

	fmt.Fprintf(conn, "DESC|it is a fruit\n")

	select {
	case evt := <-srv.OnDescMsg:
		if evt.PlayerName != "Alice" {
			t.Errorf("expected player Alice, got %s", evt.PlayerName)
		}
		if evt.Description != "it is a fruit" {
			t.Errorf("expected description %q, got %q", "it is a fruit", evt.Description)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for DescEvent")
	}
}

func TestVote_UnnamedPlayer(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send VOTE without joining — should get ERROR.
	fmt.Fprintf(conn, "VOTE|Bob\n")

	msg := readMsg(t, conn)
	if msg.Type != game.MsgError {
		t.Errorf("expected ERROR for unnamed VOTE, got %s", msg.Type)
	}
	if msg.Payload != "请先加入游戏" {
		t.Errorf("unexpected error payload: %q", msg.Payload)
	}
}

func TestSetAndVerifyRoomHash(t *testing.T) {
	srv, _ := startTestServer(t)
	defer srv.Stop()

	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	srv.SetRoomHash(hash)

	if !srv.VerifyRoomHash(hash) {
		t.Error("VerifyRoomHash should return true for the correct hash")
	}
}

func TestVerifyRoomHash_WrongHash(t *testing.T) {
	srv, _ := startTestServer(t)
	defer srv.Stop()

	srv.SetRoomHash("correct_hash")
	if srv.VerifyRoomHash("wrong_hash") {
		t.Error("VerifyRoomHash should return false for a wrong hash")
	}
}

func TestVerifyRoomHash_EmptyHash(t *testing.T) {
	srv, _ := startTestServer(t)
	defer srv.Stop()

	// Without setting any hash, roomHash is empty
	if srv.VerifyRoomHash("anything") {
		t.Error("VerifyRoomHash should return false when no room hash is set")
	}

	// Also verify that setting to empty string still returns false
	srv.SetRoomHash("")
	if srv.VerifyRoomHash("anything") {
		t.Error("VerifyRoomHash should return false when room hash is set to empty string")
	}
}

func TestSetRoomHash_Overwrite(t *testing.T) {
	srv, _ := startTestServer(t)
	defer srv.Stop()

	srv.SetRoomHash("hash1")
	srv.SetRoomHash("hash2")

	if srv.VerifyRoomHash("hash1") {
		t.Error("VerifyRoomHash should return false for old hash after overwrite")
	}
	if !srv.VerifyRoomHash("hash2") {
		t.Error("VerifyRoomHash should return true for the new hash after overwrite")
	}
}

func TestSetAndVerifyRoomHash_ConcurrentSafe(t *testing.T) {
	srv, _ := startTestServer(t)
	defer srv.Stop()

	const goroutines = 20
	done := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			hash := fmt.Sprintf("hash_%d", n)
			srv.SetRoomHash(hash)
			srv.VerifyRoomHash(hash)
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for goroutines")
		}
	}
}

func TestVote_NamedPlayerForwarded(t *testing.T) {
	srv, port := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "JOIN|Alice\n")
	readMsg(t, conn) // consume JOIN confirmation

	fmt.Fprintf(conn, "VOTE|Bob\n")

	select {
	case evt := <-srv.OnVoteMsg:
		if evt.PlayerName != "Alice" {
			t.Errorf("expected player Alice, got %s", evt.PlayerName)
		}
		if evt.Target != "Bob" {
			t.Errorf("expected target Bob, got %s", evt.Target)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for VoteEvent")
	}
}
