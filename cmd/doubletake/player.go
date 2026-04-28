package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/smallnest/doubletake/client"
	"github.com/smallnest/doubletake/game"
)

// roleDisplayNames maps internal role names to user-facing display labels.
var roleDisplayNames = map[string]string{
	"Civilian":   "平民",
	"Undercover": "卧底",
	"Blank":      "白板",
}

// descPhase tracks the player's state during the description phase.
type descPhase int

const (
	descIdle       descPhase = iota // not in desc phase or waiting for other players
	descWaitingInput                // waiting for user to type description
	descSubmitted                   // description submitted, waiting for server response
)

func RunPlayer(out io.Writer, in io.Reader, stealth bool) int {
	disp := client.NewDisplay(out, stealth)
	disp.PrintStartup()

	scanner := bufio.NewScanner(in)

	disp.Info("0000", "input room code")
	fmt.Fprint(out, "  room code: ")
	if !scanner.Scan() {
		return 1
	}
	roomCode := strings.TrimSpace(scanner.Text())
	if roomCode == "" {
		disp.Warn("room code cannot be empty")
		return 1
	}

	addr, err := game.DecodeRoomCode(roomCode)
	if err != nil {
		disp.Warn(fmt.Sprintf("invalid room code: %v", err))
		return 1
	}

	cc := client.NewClient()
	if err := cc.Connect(addr); err != nil {
		disp.Warn(fmt.Sprintf("connection failed: %v", err))
		return 1
	}
	defer cc.Disconnect()

	disp.Info("0000", "input player name")
	fmt.Fprint(out, "  name: ")
	if !scanner.Scan() {
		return 1
	}
	playerName := strings.TrimSpace(scanner.Text())
	if playerName == "" {
		disp.Warn("name cannot be empty")
		return 1
	}

	if err := cc.Send(game.Message{Type: game.MsgJoin, Payload: playerName}); err != nil {
		disp.Warn(fmt.Sprintf("send failed: %v", err))
		return 1
	}

	// phase tracks the description phase state.
	phase := descIdle

	// stdinCh is lazily initialized when first needed
	// (when we enter descWaitingInput during the desc phase).
	var stdinCh <-chan string

	for {
		// If we need stdin input, start the reader if not already running.
		if phase == descWaitingInput && stdinCh == nil {
			ch := make(chan string, 1)
			stdinCh = ch
			go func() {
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					ch <- line
				}
			}()
		}

		if phase == descWaitingInput {
			// Select between network messages and stdin input.
			select {
			case msg, ok := <-cc.Messages():
				if !ok {
					return 0
				}
				if !handleMessage(msg, disp, out, cc, playerName, &phase) {
					return 1
				}
			case line, ok := <-stdinCh:
				if !ok {
					return 0
				}
				if line == "" {
					fmt.Fprintln(out, "  描述不能为空")
					fmt.Fprint(out, "  请输入描述: ")
					continue
				}
				if err := cc.Send(game.Message{Type: game.MsgDesc, Payload: line}); err != nil {
					disp.Warn(fmt.Sprintf("send failed: %v", err))
					return 1
				}
				phase = descSubmitted
			}
		} else {
			// Simple blocking read from network messages only.
			msg, ok := <-cc.Messages()
			if !ok {
				return 0
			}
			if !handleMessage(msg, disp, out, cc, playerName, &phase) {
				return 1
			}
		}
	}
}

// handleMessage processes a single network message. It returns false if the
// player should exit (fatal error).
func handleMessage(msg game.Message, disp *client.Display, out io.Writer, cc *client.Client, playerName string, phase *descPhase) bool {
	switch msg.Type {
	case game.MsgJoin:
		disp.Info("0000", fmt.Sprintf("joined as %s", msg.Payload))
	case game.MsgRole:
		parts := strings.SplitN(msg.Payload, "|", 2)
		if len(parts) < 2 {
			disp.Data("00", "received malformed role message")
			return true
		}
		roleName, word := parts[0], parts[1]
		dispLabel := roleName
		if label, ok := roleDisplayNames[roleName]; ok {
			dispLabel = label
		}
		if roleName == "Blank" {
			disp.Data("00", fmt.Sprintf("assigned token: [%s] — 你是白板", dispLabel))
		} else {
			disp.Data("00", fmt.Sprintf("assigned token: %s [%s]", word, dispLabel))
		}
	case game.MsgRound:
		handleRoundMsg(disp, msg.Payload)
	case game.MsgTurn:
		speaker := msg.Payload
		if speaker == playerName {
			// Our turn — prompt for input
			fmt.Fprint(out, "  请输入描述: ")
			*phase = descWaitingInput
		} else {
			disp.Data("00", fmt.Sprintf("等待 %s 描述...", speaker))
			*phase = descIdle
		}
	case game.MsgDesc:
		handleDescMsg(disp, msg.Payload)
		// If we were in descSubmitted, our description was accepted
		if *phase == descSubmitted {
			*phase = descIdle
		}
	case game.MsgError:
		disp.Warn(msg.Payload)
		if *phase == descSubmitted || *phase == descWaitingInput {
			// In desc phase: re-prompt for input
			fmt.Fprint(out, "  请输入描述: ")
			*phase = descWaitingInput
		} else {
			return false
		}
	default:
		disp.Data("00", fmt.Sprintf("%s %s", msg.Type, msg.Payload))
	}
	return true
}

// handleRoundMsg parses and displays the ROUND message.
// Payload format: "roundNum|P0,P1,P2"
func handleRoundMsg(disp *client.Display, payload string) {
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) < 2 {
		disp.Data("00", fmt.Sprintf("ROUND %s", payload))
		return
	}
	roundNum := parts[0]
	speakers := strings.Split(parts[1], ",")
	orderStr := strings.Join(speakers, " → ")
	disp.Data("00", fmt.Sprintf("轮次 %s，发言顺序: %s", roundNum, orderStr))
}

// handleDescMsg parses and displays a DESC broadcast.
// Payload format: "playerName|description"
func handleDescMsg(disp *client.Display, payload string) {
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) < 2 {
		disp.Data("00", fmt.Sprintf("DESC %s", payload))
		return
	}
	playerName := parts[0]
	desc := parts[1]
	disp.Data("00", fmt.Sprintf("%s: %s", playerName, desc))
}
