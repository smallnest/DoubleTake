package server

import (
	"bufio"
	"fmt"
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
	gamePlayers  map[string]*game.Player // name → game player (persists across disconnects)
	mu           sync.Mutex
	done         chan struct{}
	stopOnce     sync.Once
	ready        chan struct{}
	OnPlayerJoin chan PlayerJoinEvent  // notifies when a named player joins
	OnDescMsg    chan DescEvent        // forwards DESC messages from named players
	OnVoteMsg    chan VoteEvent        // forwards VOTE messages from named players
	OnGuessMsg   chan GuessEvent       // forwards GUESS messages from named players
	OnDisconnect chan DisconnectEvent  // notifies when a named player disconnects
}

// PlayerJoinEvent carries info about a player that just joined.
type PlayerJoinEvent struct {
	Name      string
	Current   int // number of named players after this join
	Capacity  int // configured totalPlayers
}

// DescEvent carries a DESC message received from a named player.
type DescEvent struct {
	PlayerName  string
	Description string
}

// VoteEvent carries a VOTE message received from a named player.
type VoteEvent struct {
	PlayerName string
	Target     string
}

// GuessEvent carries a GUESS message received from a named player.
type GuessEvent struct {
	PlayerName string
	Word       string
}

// DisconnectEvent carries info about a player that just disconnected.
type DisconnectEvent struct {
	PlayerName string
}

// NewServer creates a new Server that will listen on the given port.
// totalPlayers sets the expected player capacity for join notifications.
func NewServer(port string, totalPlayers int) *Server {
	return &Server{
		port:         port,
		totalPlayers: totalPlayers,
		connections:  make(map[net.Conn]*Player),
		names:        make(map[string]bool),
		gamePlayers:  make(map[string]*game.Player),
		done:         make(chan struct{}),
		ready:        make(chan struct{}),
		OnPlayerJoin: make(chan PlayerJoinEvent, 64),
		OnDescMsg:    make(chan DescEvent, 64),
		OnVoteMsg:    make(chan VoteEvent, 64),
		OnGuessMsg:   make(chan GuessEvent, 64),
		OnDisconnect: make(chan DisconnectEvent, 64),
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
		case game.MsgReconnect:
			s.handleReconnect(player, msg.Payload)
		case game.MsgDesc:
			s.handleDesc(player, msg.Payload)
		case game.MsgVote:
			s.handleVote(player, msg.Payload)
		case game.MsgGuess:
			s.handleGuess(player, msg.Payload)
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
		// Named player: keep name reserved, mark game player as disconnected.
		if gp, ok := s.gamePlayers[player.Name]; ok {
			gp.Connected = false
		}
		s.mu.Unlock()
		player.Conn.Close()
		log.Printf("[WARN] %s disconnected", player.Name)

		select {
		case s.OnDisconnect <- DisconnectEvent{PlayerName: player.Name}:
		default:
		}
	} else {
		// Unnamed player: full cleanup.
		s.mu.Unlock()
		player.Conn.Close()
		log.Printf("player disconnected: %s", player.Conn.RemoteAddr())
	}
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
		// Check if this is a disconnected player's name.
		if gp, ok := s.gamePlayers[name]; ok && !gp.Connected {
			s.mu.Unlock()
			s.Send(player.Conn, game.Message{Type: game.MsgError, Payload: "该玩家已掉线，请使用 RECONNECT 重连"})
			return
		}
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

// SendToPlayer sends a message to a player by name.
// Returns an error if the player is not found.
func (s *Server) SendToPlayer(name string, msg game.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, player := range s.connections {
		if player.Name == name {
			data := []byte(game.Encode(msg))
			if _, err := player.Conn.Write(data); err != nil {
				return fmt.Errorf("send to player %s: %w", name, err)
			}
			return nil
		}
	}
	return fmt.Errorf("player %s not found", name)
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

// handleDesc processes a DESC message from a player.
// Unnamed players receive an error. Named players' descriptions are forwarded
// to the OnDescMsg channel for the judge to validate and process.
func (s *Server) handleDesc(player *Player, payload string) {
	if player.Name == "" {
		s.Send(player.Conn, game.Message{Type: game.MsgError, Payload: "请先加入游戏"})
		return
	}

	select {
	case s.OnDescMsg <- DescEvent{PlayerName: player.Name, Description: payload}:
	default:
		log.Printf("OnDescMsg channel full, dropping DESC from %s", player.Name)
	}
}

// handleVote processes a VOTE message from a player.
// Unnamed players receive an error. Named players' votes are forwarded
// to the OnVoteMsg channel for the judge to validate and process.
func (s *Server) handleVote(player *Player, payload string) {
	if player.Name == "" {
		s.Send(player.Conn, game.Message{Type: game.MsgError, Payload: "请先加入游戏"})
		return
	}

	select {
	case s.OnVoteMsg <- VoteEvent{PlayerName: player.Name, Target: payload}:
	default:
		log.Printf("OnVoteMsg channel full, dropping VOTE from %s", player.Name)
	}
}

// handleGuess processes a GUESS message from a player.
// Unnamed players receive an error. Named players' guesses are forwarded
// to the OnGuessMsg channel for the judge to validate and process.
func (s *Server) handleGuess(player *Player, payload string) {
	if player.Name == "" {
		s.Send(player.Conn, game.Message{Type: game.MsgError, Payload: "请先加入游戏"})
		return
	}

	select {
	case s.OnGuessMsg <- GuessEvent{PlayerName: player.Name, Word: payload}:
	default:
		log.Printf("OnGuessMsg channel full, dropping GUESS from %s", player.Name)
	}
}

// SetGamePlayers registers the game player objects with the server.
// This must be called after game.AssignRoles so the server can track
// player state across disconnects and reconnects.
func (s *Server) SetGamePlayers(players []*game.Player) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range players {
		s.gamePlayers[p.Name] = p
		p.Connected = true
	}
}

// handleReconnect processes a RECONNECT message from a disconnected player.
// The payload should be the player's name. If the name matches a disconnected
// player, the connection is restored; otherwise an ERROR is returned.
func (s *Server) handleReconnect(player *Player, name string) {
	if name == "" {
		s.Send(player.Conn, game.Message{Type: game.MsgError, Payload: "名字不能为空"})
		return
	}

	s.mu.Lock()
	gp, ok := s.gamePlayers[name]
	if !ok || gp.Connected {
		s.mu.Unlock()
		s.Send(player.Conn, game.Message{Type: game.MsgError, Payload: "重连失败：名字不匹配或玩家未掉线"})
		return
	}

	// Restore connection.
	player.Name = name
	gp.Connected = true
	s.connections[player.Conn] = player
	s.mu.Unlock()

	s.Send(player.Conn, game.Message{Type: game.MsgReconnect, Payload: name})
	log.Printf("player %s reconnected as %s", player.Conn.RemoteAddr(), name)
}
