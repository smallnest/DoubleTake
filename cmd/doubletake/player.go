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

	for msg := range cc.Messages() {
		switch msg.Type {
		case game.MsgJoin:
			disp.Info("0000", fmt.Sprintf("joined as %s", msg.Payload))
		case game.MsgRole:
			parts := strings.SplitN(msg.Payload, "|", 2)
			if len(parts) < 2 {
				disp.Data("00", "received malformed role message")
				continue
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
		case game.MsgError:
			disp.Warn(msg.Payload)
			return 1
		default:
			disp.Data("00", fmt.Sprintf("%s %s", msg.Type, msg.Payload))
		}
	}

	return 0
}
