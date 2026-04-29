package game

import (
	"errors"
	"strings"
)

// ErrInvalidMessage indicates the message format is invalid.
var ErrInvalidMessage = errors.New("invalid message format")

// Encode converts a Message to the wire format "TYPE|payload\n".
func Encode(msg Message) string {
	return msg.Type + "|" + msg.Payload + "\n"
}

// Decode parses a "TYPE|payload\n" string into a Message.
// It only splits on the first '|' so payloads may contain '|'.
func Decode(raw string) (Message, error) {
	line := strings.TrimRight(raw, "\n")
	if line == "" {
		return Message{}, ErrInvalidMessage
	}
	parts := strings.SplitN(line, "|", 2)
	if len(parts) < 2 {
		return Message{}, ErrInvalidMessage
	}
	return Message{Type: parts[0], Payload: parts[1]}, nil
}

// Message types for the communication protocol between referee and players.
const (
	MsgJoin      = "JOIN"
	MsgReady     = "READY"
	MsgRole      = "ROLE"
	MsgTurn      = "TURN"
	MsgDesc      = "DESC"
	MsgVote      = "VOTE"
	MsgResult    = "RESULT"
	MsgKick      = "KICK"
	MsgWin       = "WIN"
	MsgError     = "ERROR"
	MsgReconnect = "RECONNECT"
	MsgGuess     = "GUESS"
	MsgStart     = "START"
	MsgPlayers   = "PLAYERS"
	MsgPKStart   = "PK_START"
	MsgPKVote    = "PK_VOTE"
	MsgState     = "STATE"
	MsgRestart   = "RESTART"
	MsgQuit      = "QUIT"
	MsgRound     = "ROUND"
	MsgWord           = "WORD"
	MsgVoteBroadcast  = "VOTE_BC"
)

// Message represents a protocol message exchanged between referee and player.
type Message struct {
	Type    string
	Payload string
}
