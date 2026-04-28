package server

import (
	"bufio"
	"log"
	"net"
	"sync"

	"github.com/smallnest/doubletake/game"
)

// Player represents a connected client in the game.
type Player struct {
	Conn net.Conn
	Name string
}

// Server is a TCP server that manages client connections for the game.
type Server struct {
	port         string
	totalPlayers int                   // configured player capacity
	listener     net.Listener
	connections  map[net.Conn]*Player
	names        map[string]bool
	mu           sync.Mutex
	done         chan struct{}
	stopOnce     sync.Once
	ready        chan struct{}
	OnPlayerJoin chan PlayerJoinEvent // notifies when a named player joins
}

// PlayerJoinEvent carries info about a player that just joined.
type PlayerJoinEvent struct {
	Name      string
	Current   int // number of named players after this join
	Capacity  int // configured totalPlayers
}

// NewServer creates a new Server that will listen on the given port.
// totalPlayers sets the expected player capacity for join notifications.
func NewServer(port string, totalPlayers int) *Server {
	return &Server{
		port:         port,
		totalPlayers: totalPlayers,
		connections:  make(map[net.Conn]*Player),
		names:        make(map[string]bool),
		done:         make(chan struct{}),
		ready:        make(chan struct{}),
		OnPlayerJoin: make(chan PlayerJoinEvent, 64),
	}
}

// Start begins listening on the TCP port and accepting connections.
// This method blocks; call it in a goroutine if non-blocking behavior is needed.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		close(s.ready) // unblock Stop() even on listen failure
		return err
	}
	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()
	log.Printf("server listening on :%s", s.port)
	close(s.ready)

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

// Stop closes the listener and all active client connections.
// It is safe to call Stop() multiple times.
func (s *Server) Stop() {
	<-s.ready // wait for Start() to complete listener setup

	s.stopOnce.Do(func() {
		close(s.done)

		s.mu.Lock()
		if s.listener != nil {
			s.listener.Close()
		}
		// Collect connections and clear the map while holding the lock,
		// then close connections outside the lock to avoid deadlock
		// with unregister() which also acquires s.mu.
		conns := make([]net.Conn, 0, len(s.connections))
		for conn := range s.connections {
			conns = append(conns, conn)
		}
		s.connections = make(map[net.Conn]*Player)
		s.mu.Unlock()

		for _, conn := range conns {
			conn.Close()
		}
		log.Println("server stopped")
	})
}

// handleConn reads messages from a client connection and processes them.
func (s *Server) handleConn(conn net.Conn) {
	player := &Player{Conn: conn}
	s.register(player)
	defer s.unregister(player)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		msg, err := game.Decode(line)
		if err != nil {
			log.Printf("decode error from %s: %v", conn.RemoteAddr(), err)
			continue
		}
		log.Printf("received from %s: %s %s", conn.RemoteAddr(), msg.Type, msg.Payload)

		switch msg.Type {
		case game.MsgJoin:
			s.handleJoin(player, msg.Payload)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("connection error from %s: %v", conn.RemoteAddr(), err)
	}
}

func (s *Server) register(player *Player) {
	s.mu.Lock()
	s.connections[player.Conn] = player
	s.mu.Unlock()
	log.Printf("player connected: %s", player.Conn.RemoteAddr())
}

func (s *Server) unregister(player *Player) {
	s.mu.Lock()
	delete(s.connections, player.Conn)
	if player.Name != "" {
		delete(s.names, player.Name)
	}
	s.mu.Unlock()
	player.Conn.Close()
	log.Printf("player disconnected: %s", player.Conn.RemoteAddr())
}

// Broadcast sends a message to all connected players.
func (s *Server) Broadcast(msg game.Message) {
	data := []byte(game.Encode(msg))
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.connections {
		if _, err := conn.Write(data); err != nil {
			log.Printf("broadcast write error to %s: %v", conn.RemoteAddr(), err)
		}
	}
}

func (s *Server) handleJoin(player *Player, name string) {
	if name == "" {
		s.Send(player.Conn, game.Message{Type: game.MsgError, Payload: "名字不能为空"})
		return
	}

	s.mu.Lock()
	if s.names[name] {
		s.mu.Unlock()
		s.Send(player.Conn, game.Message{Type: game.MsgError, Payload: "名字已存在，请换一个"})
		return
	}
	player.Name = name
	s.names[name] = true
	count := len(s.names)
	capacity := s.totalPlayers
	s.mu.Unlock()

	s.Send(player.Conn, game.Message{Type: game.MsgJoin, Payload: name})
	log.Printf("player %s joined as %s", player.Conn.RemoteAddr(), name)

	// Notify listeners about the new player join (non-blocking).
	select {
	case s.OnPlayerJoin <- PlayerJoinEvent{Name: name, Current: count, Capacity: capacity}:
	default:
	}
}

// Send sends a message to a single connection.
func (s *Server) Send(conn net.Conn, msg game.Message) {
	data := []byte(game.Encode(msg))
	if _, err := conn.Write(data); err != nil {
		log.Printf("send write error to %s: %v", conn.RemoteAddr(), err)
	}
}

// PlayerCount returns the number of named (registered) players.
func (s *Server) PlayerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.names)
}

// PlayerNames returns the names of all registered players.
func (s *Server) PlayerNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.names))
	for name := range s.names {
		names = append(names, name)
	}
	return names
}

// BroadcastToNamedPlayers sends a message to all connections that have
// completed the JOIN handshake (i.e. have a non-empty Name).
func (s *Server) BroadcastToNamedPlayers(msg game.Message) {
	data := []byte(game.Encode(msg))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, player := range s.connections {
		if player.Name == "" {
			continue
		}
		if _, err := player.Conn.Write(data); err != nil {
			log.Printf("broadcast write error to %s: %v", player.Conn.RemoteAddr(), err)
		}
	}
}
