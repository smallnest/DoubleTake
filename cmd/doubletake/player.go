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

// votePhase tracks the player's state during the voting phase.
type votePhase int

const (
	voteIdle       votePhase = iota // not in vote phase or waiting for other players
	voteWaitingInput                // waiting for user to type vote target
	voteSubmitted                   // vote submitted, waiting for server response
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

	// descP tracks the description phase state; voteP tracks the voting phase state.
	descP := descIdle
	voteP := voteIdle
	// inVotePhase is set to true when a VOTE message is received,
	// indicating we have entered the voting phase.
	inVotePhase := false

	// stdinCh is lazily initialized when first needed
	// (when we enter a waiting-input state during either phase).
	var stdinCh <-chan string

	for {
		waitingInput := descP == descWaitingInput || voteP == voteWaitingInput

		// If we need stdin input, start the reader if not already running.
		if waitingInput && stdinCh == nil {
			ch := make(chan string, 1)
			stdinCh = ch
			go func() {
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					ch <- line
				}
			}()
		}

		if waitingInput {
			// Select between network messages and stdin input.
			select {
			case msg, ok := <-cc.Messages():
				if !ok {
					return 0
				}
				if code := handleMessage(msg, disp, out, cc, playerName, &descP, &voteP, &inVotePhase); code >= 0 {
					return code
				}
			case line, ok := <-stdinCh:
				if !ok {
					return 0
				}
				if descP == descWaitingInput {
					if line == "" {
						fmt.Fprintln(out, "  描述不能为空")
						fmt.Fprint(out, "  请输入描述: ")
						continue
					}
					if err := cc.Send(game.Message{Type: game.MsgDesc, Payload: line}); err != nil {
						disp.Warn(fmt.Sprintf("send failed: %v", err))
						return 1
					}
					descP = descSubmitted
				} else if voteP == voteWaitingInput {
					if line == "" {
						fmt.Fprintln(out, "  投票目标不能为空")
						fmt.Fprint(out, "  请输入投票目标: ")
						continue
					}
					if err := cc.Send(game.Message{Type: game.MsgVote, Payload: line}); err != nil {
						disp.Warn(fmt.Sprintf("send failed: %v", err))
						return 1
					}
					voteP = voteSubmitted
				}
			}
		} else {
			// Simple blocking read from network messages only.
			msg, ok := <-cc.Messages()
			if !ok {
				return 0
			}
			if code := handleMessage(msg, disp, out, cc, playerName, &descP, &voteP, &inVotePhase); code >= 0 {
				return code
			}
		}
	}
}

// handleMessage processes a single network message.
// Returns: -1 = continue, 0 = exit normally (e.g. WIN), 1 = exit with error.
func handleMessage(msg game.Message, disp *client.Display, out io.Writer, cc *client.Client, playerName string, descP *descPhase, voteP *votePhase, inVotePhase *bool) int {
	switch msg.Type {
	case game.MsgJoin:
		disp.Info("0000", fmt.Sprintf("joined as %s", msg.Payload))
	case game.MsgRole:
		parts := strings.SplitN(msg.Payload, "|", 2)
		if len(parts) < 2 {
			disp.Data("00", "received malformed role message")
			return -1
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
	case game.MsgVote:
		handleVoteMsg(disp, msg.Payload)
		*voteP = voteIdle
		*inVotePhase = true
	case game.MsgTurn:
		speaker := msg.Payload
		if *inVotePhase {
			// In voting phase
			if speaker == playerName {
				fmt.Fprint(out, "  请输入投票目标: ")
				*voteP = voteWaitingInput
			} else {
				disp.Data("00", fmt.Sprintf("等待 %s 投票...", speaker))
				*voteP = voteIdle
			}
		} else {
			// In description phase (or pre-phase)
			if speaker == playerName {
				fmt.Fprint(out, "  请输入描述: ")
				*descP = descWaitingInput
			} else {
				disp.Data("00", fmt.Sprintf("等待 %s 描述...", speaker))
				*descP = descIdle
			}
		}
	case game.MsgDesc:
		handleDescMsg(disp, msg.Payload)
		// If we were in descSubmitted, our description was accepted
		if *descP == descSubmitted {
			*descP = descIdle
		}
	case game.MsgPKStart:
		handlePKStartMsg(disp, msg.Payload)
		// Reset desc phase for PK descriptions.
		*descP = descIdle
		*voteP = voteIdle
		*inVotePhase = false
	case game.MsgPKVote:
		handlePKVoteMsg(disp, msg.Payload)
		*voteP = voteIdle
		*inVotePhase = true
	case game.MsgResult:
		handleResultMsg(disp, msg.Payload)
	case game.MsgWin:
		handleWinMsg(disp, msg.Payload)
		return 0
	case game.MsgError:
		disp.Warn(msg.Payload)
		if *descP == descSubmitted || *descP == descWaitingInput {
			// In desc phase: re-prompt for input
			fmt.Fprint(out, "  请输入描述: ")
			*descP = descWaitingInput
		} else if *voteP == voteSubmitted || *voteP == voteWaitingInput {
			// In vote phase: re-prompt for input
			fmt.Fprint(out, "  请输入投票目标: ")
			*voteP = voteWaitingInput
		} else {
			return 1
		}
	default:
		disp.Data("00", fmt.Sprintf("%s %s", msg.Type, msg.Payload))
	}
	return -1
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

// handleVoteMsg parses and displays the VOTE broadcast.
// Payload format: "roundNum|alivePlayerList"
func handleVoteMsg(disp *client.Display, payload string) {
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) < 2 {
		disp.Data("00", fmt.Sprintf("VOTE %s", payload))
		return
	}
	roundNum := parts[0]
	playerList := strings.Split(parts[1], ",")
	orderStr := strings.Join(playerList, " → ")
	disp.Data("00", fmt.Sprintf("投票环节 轮次 %s，可投票: %s", roundNum, orderStr))
}

// handleResultMsg parses and displays the RESULT broadcast.
// Payload format: "playerA:count,playerB:count,..."
func handleResultMsg(disp *client.Display, payload string) {
	if payload == "" {
		disp.Data("00", "投票结果: 无")
		return
	}
	pairs := strings.Split(payload, ",")
	disp.Data("00", "投票结果:")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) < 2 {
			disp.Data("00", fmt.Sprintf("  %s", pair))
			continue
		}
		disp.Data("00", fmt.Sprintf("  %s: %s 票", kv[0], kv[1]))
	}
}

// handleWinMsg parses and displays the WIN message.
// Payload format: "winner|player1:Role:alive,player2:Role:alive,...|civilianWord|undercoverWord"
func handleWinMsg(disp *client.Display, payload string) {
	parts := strings.SplitN(payload, "|", 4)
	if len(parts) < 4 {
		disp.Data("00", fmt.Sprintf("WIN %s", payload))
		return
	}
	winner := parts[0]
	statesStr := parts[1]
	civilianWord := parts[2]
	undercoverWord := parts[3]

	winnerLabel := winner
	if label, ok := roleDisplayNames[winner]; ok {
		winnerLabel = label
	}

	var results []client.PlayerResult
	for _, state := range strings.Split(statesStr, ",") {
		sp := strings.SplitN(state, ":", 3)
		if len(sp) < 3 {
			continue
		}
		name, roleName, aliveStr := sp[0], sp[1], sp[2]
		roleLabel := roleName
		if label, ok := roleDisplayNames[roleName]; ok {
			roleLabel = label
		}
		results = append(results, client.PlayerResult{
			Name:  name,
			Role:  roleLabel,
			Alive: aliveStr == "1",
		})
	}
	disp.ShowGameResult(winnerLabel, results, civilianWord, undercoverWord)
}

// handlePKStartMsg parses and displays the PK_START broadcast.
// Payload format: "pkNum|tiedPlayerList"
func handlePKStartMsg(disp *client.Display, payload string) {
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) < 2 {
		disp.Data("00", fmt.Sprintf("PK %s", payload))
		return
	}
	pkNum := parts[0]
	tiedPlayers := strings.Split(parts[1], ",")
	orderStr := strings.Join(tiedPlayers, " → ")
	disp.Data("00", fmt.Sprintf("平票！PK 第 %s 轮，PK 玩家: %s", pkNum, orderStr))
}

// handlePKVoteMsg parses and displays the PK_VOTE broadcast.
// Payload format: "pkNum|tiedPlayerList"
func handlePKVoteMsg(disp *client.Display, payload string) {
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) < 2 {
		disp.Data("00", fmt.Sprintf("PK_VOTE %s", payload))
		return
	}
	pkNum := parts[0]
	tiedPlayers := strings.Split(parts[1], ",")
	orderStr := strings.Join(tiedPlayers, " → ")
	disp.Data("00", fmt.Sprintf("PK 投票环节 第 %s 轮，可投票: %s", pkNum, orderStr))
}
