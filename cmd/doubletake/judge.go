package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/smallnest/doubletake/client"
	"github.com/smallnest/doubletake/game"
	"github.com/smallnest/doubletake/server"
)

// descResult holds the outcome of the description phase.
type descResult struct {
	Round *game.DescRound
}

// GameConfig holds the validated game parameters entered by the referee.
type GameConfig struct {
	TotalPlayers int // 总人数 (4-10)
	Undercovers  int // 卧底人数 (1-3)
	Blanks       int // 白板人数 (0+)
}

// readInt reads a line from the scanner, trims whitespace, and parses it as an int.
func readInt(scanner *bufio.Scanner) (int, error) {
	if !scanner.Scan() {
		return 0, fmt.Errorf("unexpected end of input")
	}
	line := strings.TrimSpace(scanner.Text())
	return strconv.Atoi(line)
}

// collectConfig interactively prompts the referee for game parameters and validates them.
// It loops on invalid input, printing specific error reasons via the Display.
func collectConfig(out io.Writer, disp *client.Display, scanner *bufio.Scanner) GameConfig {
	for {
		disp.Info("0000", "输入游戏参数")

		fmt.Fprint(out, "  玩家人数 (4-10): ")
		total := readIntInput(scanner, disp)

		fmt.Fprint(out, "  卧底人数 (1-3): ")
		undercovers := readIntInput(scanner, disp)

		fmt.Fprint(out, "  白板人数 (0+): ")
		blanks := readIntInput(scanner, disp)

		if err := validateConfig(total, undercovers, blanks); err != nil {
			disp.Warn(err.Error())
			continue
		}

		return GameConfig{
			TotalPlayers: total,
			Undercovers:  undercovers,
			Blanks:       blanks,
		}
	}
}

// readIntInput reads an integer from the scanner.
// On parse failure it prints a warning and returns -1 so validation will fail.
func readIntInput(scanner *bufio.Scanner, disp *client.Display) int {
	n, err := readInt(scanner)
	if err != nil {
		disp.Warn("请输入有效的数字")
		return -1
	}
	return n
}

// validateConfig checks game parameters and returns a descriptive error if invalid.
func validateConfig(total, undercovers, blanks int) error {
	if total < 4 || total > 10 {
		return fmt.Errorf("玩家人数 %d 不在合法范围 (4-10)，请重新输入", total)
	}
	if undercovers < 1 || undercovers > 3 {
		return fmt.Errorf("卧底人数 %d 不在合法范围 (1-3)，请重新输入", undercovers)
	}
	if blanks < 0 {
		return fmt.Errorf("白板人数 %d 不能为负数，请重新输入", blanks)
	}
	if undercovers+blanks >= (total+1)/2 {
		return fmt.Errorf("卧底(%d)+白板(%d)=%d，必须少于总人数的一半(%d)，请重新输入", undercovers, blanks, undercovers+blanks, (total+1)/2)
	}
	return nil
}

// collectWords interactively prompts the referee for two words and validates they differ.
// It loops until two different words are entered.
func collectWords(out io.Writer, disp *client.Display, scanner *bufio.Scanner) (civilianWord, undercoverWord string) {
	for {
		fmt.Fprint(out, "  平民词语: ")
		if !scanner.Scan() {
			return "", ""
		}
		civilianWord = strings.TrimSpace(scanner.Text())

		fmt.Fprint(out, "  卧底词语: ")
		if !scanner.Scan() {
			return "", ""
		}
		undercoverWord = strings.TrimSpace(scanner.Text())

		if civilianWord == "" || undercoverWord == "" {
			disp.Warn("词语不能为空，请重新输入")
			continue
		}
		if civilianWord == undercoverWord {
			disp.Warn("平民词语和卧底词语不能相同，请重新输入")
			continue
		}
		return civilianWord, undercoverWord
	}
}

// collectWordsFromCh reads two words from a channel (same as collectWords but uses channel input).
// It loops until two different words are entered.
func collectWordsFromCh(out io.Writer, disp *client.Display, stdinCh <-chan string, stdinDone <-chan struct{}) (civilianWord, undercoverWord string) {
	for {
		fmt.Fprint(out, "  平民词语: ")
		var ok bool
		select {
		case civilianWord, ok = <-stdinCh:
			if !ok {
				return "", ""
			}
			civilianWord = strings.TrimSpace(civilianWord)
		case <-stdinDone:
			return "", ""
		}

		fmt.Fprint(out, "  卧底词语: ")
		select {
		case undercoverWord, ok = <-stdinCh:
			if !ok {
				return "", ""
			}
			undercoverWord = strings.TrimSpace(undercoverWord)
		case <-stdinDone:
			return "", ""
		}

		if civilianWord == "" || undercoverWord == "" {
			disp.Warn("词语不能为空，请重新输入")
			continue
		}
		if civilianWord == undercoverWord {
			disp.Warn("平民词语和卧底词语不能相同，请重新输入")
			continue
		}
		return civilianWord, undercoverWord
	}
}

// stdinSource holds the shared stdin reading channels used by waitingPhase and collectWords.
type stdinSource struct {
	ch   <-chan string
	done <-chan struct{}
}

// RunJudge runs the referee interactive configuration flow and the waiting phase.
// out is used for display output; in provides the interactive input source.
// It returns the GameConfig that was selected.
func RunJudge(out io.Writer, in io.Reader, port string, stealth bool) GameConfig {
	disp := client.NewDisplay(out, stealth)
	disp.PrintStartup()

	scanner := bufio.NewScanner(in)
	cfg := collectConfig(out, disp, scanner)

	srv := server.NewServer(port, cfg.TotalPlayers)
	go srv.Start()
	defer srv.Stop()

	disp.Info("0000", fmt.Sprintf("房间已创建，等待 %d 名玩家加入...", cfg.TotalPlayers))

	// waitingPhase blocks until the referee confirms game start.
	stdinSrc := newStdinSource(scanner)
	waitingPhase(out, in, disp, srv, cfg, stdinSrc)

	// Collect words and assign roles.
	civilianWord, undercoverWord := collectWordsFromCh(out, disp, stdinSrc.ch, stdinSrc.done)
	names := srv.PlayerNames()
	players, err := game.AssignRoles(names, cfg.Undercovers, cfg.Blanks)
	if err != nil {
		disp.Warn(fmt.Sprintf("角色分配失败: %v", err))
		return cfg
	}
	game.AssignWords(players, civilianWord, undercoverWord)
	sendRoleToPlayers(disp, srv, players)

	broadcastReady(disp, srv)

	// Run description phase for round 1 with all players.
	descResult := descriptionPhase(disp, srv, 1, names)
	if descResult == nil {
		return cfg
	}
	_ = descResult // descriptions recorded for later phases

	// Run voting phase for round 1.
	voteResult := votingPhase(disp, srv, 1, names, players)
	if voteResult == nil {
		return cfg
	}
	_ = voteResult

	return cfg
}

// newStdinSource starts a goroutine that reads from scanner and returns a stdinSource.
func newStdinSource(scanner *bufio.Scanner) *stdinSource {
	ch := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			ch <- line
		}
	}()
	return &stdinSource{ch: ch, done: done}
}

// waitingPhase displays player join events and reads stdin for "start" commands.
// It blocks until the referee confirms game start or stdin is closed.
func waitingPhase(out io.Writer, in io.Reader, disp *client.Display, srv *server.Server, cfg GameConfig, stdinSrc *stdinSource) {
	for {
		select {
		case evt := <-srv.OnPlayerJoin:
			disp.Info("0000", fmt.Sprintf("%s joined [%d/%d]", evt.Name, evt.Current, evt.Capacity))
			if evt.Current >= cfg.TotalPlayers {
				disp.Info("0000", "人已齐，输入 start 开始游戏")
			}

		case line := <-stdinSrc.ch:
			if strings.EqualFold(line, "start") {
				count := srv.PlayerCount()
				if count < 4 {
					disp.Warn(fmt.Sprintf("至少需要 4 人，当前 %d 人", count))
					continue
				}
				if count < cfg.TotalPlayers {
					fmt.Fprintf(out, "  当前 %d/%d 人，确认开始？(Y/N): ", count, cfg.TotalPlayers)
					// Wait for confirmation
					select {
					case confirm := <-stdinSrc.ch:
						if strings.EqualFold(confirm, "Y") || confirm == "y" {
							return
						}
						disp.Info("0000", "已取消，继续等待玩家...")
					case <-srv.OnPlayerJoin:
						// A player joined while waiting for confirmation; go back to main loop.
						continue
					case <-stdinSrc.done:
						return
					}
					continue
				}
				// count == cfg.TotalPlayers (or more, shouldn't happen)
				return
			}

		case <-stdinSrc.done:
			return
		}
	}
}

// descriptionPhase runs the description phase for one round.
// It broadcasts ROUND|roundNum|speakerOrder, then sends TURN|playerName to the
// first speaker and processes DESC messages until all players have spoken.
func descriptionPhase(disp *client.Display, srv *server.Server, roundNum int, alivePlayers []string) *descResult {
	round, err := game.NewDescRound(roundNum, alivePlayers)
	if err != nil {
		disp.Warn(fmt.Sprintf("创建描述轮次失败: %v", err))
		return nil
	}

	// Broadcast ROUND|roundNum|speakerOrder to all named players.
	speakerList := strings.Join(round.SpeakerOrder, ",")
	srv.BroadcastToNamedPlayers(game.Message{
		Type:    game.MsgRound,
		Payload: fmt.Sprintf("%d|%s", roundNum, speakerList),
	})

	// Send TURN|playerName to the first speaker.
	srv.BroadcastToNamedPlayers(game.Message{
		Type:    game.MsgTurn,
		Payload: round.CurrentSpeaker(),
	})

	for !round.AllDone() {
		evt := <-srv.OnDescMsg
		current := round.CurrentSpeaker()

		// If all done (shouldn't happen due to loop condition), break.
		if current == "" {
			break
		}

		// Check if it's this player's turn.
		if evt.PlayerName != current {
			_ = srv.SendToPlayer(evt.PlayerName, game.Message{
				Type:    game.MsgError,
				Payload: "还没轮到你发言",
			})
			continue
		}

		// Try to record the description.
		err := round.RecordDesc(evt.PlayerName, evt.Description)
		if err == game.ErrEmptyDesc {
			_ = srv.SendToPlayer(evt.PlayerName, game.Message{
				Type:    game.MsgError,
				Payload: "描述不能为空，请重新输入",
			})
			continue
		}
		// ErrNotYourTurn shouldn't happen here since we checked above, but handle defensively.
		if err != nil {
			_ = srv.SendToPlayer(evt.PlayerName, game.Message{
				Type:    game.MsgError,
				Payload: err.Error(),
			})
			continue
		}

		// Valid description: broadcast DESC|playerName|description to all.
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgDesc,
			Payload: evt.PlayerName + "|" + evt.Description,
		})

		// Send TURN to the next speaker, if any.
		if !round.AllDone() {
			srv.BroadcastToNamedPlayers(game.Message{
				Type:    game.MsgTurn,
				Payload: round.CurrentSpeaker(),
			})
		}
	}

	return &descResult{Round: round}
}

// voteResult holds the outcome of the voting phase.
type voteResult struct {
	Round      *game.VoteRound
	Eliminated string // name of the player with the most votes (empty if tie)
}

// votingPhase runs the voting phase for one round.
// It broadcasts VOTE|roundNum|alivePlayerList, sends TURN|playerName to the
// first voter, processes VOTE messages, and broadcasts RESULT when all votes are in.
// It returns the eliminated player name (empty if tie) or nil on failure.
func votingPhase(disp *client.Display, srv *server.Server, roundNum int, alivePlayers []string, players []*game.Player) *voteResult {
	round, err := game.NewVoteRound(roundNum, alivePlayers)
	if err != nil {
		disp.Warn(fmt.Sprintf("创建投票轮次失败: %v", err))
		return nil
	}

	// Broadcast VOTE|roundNum|alivePlayerList to all named players.
	playerList := strings.Join(alivePlayers, ",")
	srv.BroadcastToNamedPlayers(game.Message{
		Type:    game.MsgVote,
		Payload: fmt.Sprintf("%d|%s", roundNum, playerList),
	})

	// Send TURN|playerName to the first voter.
	srv.BroadcastToNamedPlayers(game.Message{
		Type:    game.MsgTurn,
		Payload: round.CurrentVoter(),
	})

	for !round.AllDone() {
		evt := <-srv.OnVoteMsg
		current := round.CurrentVoter()

		if current == "" {
			break
		}

		// Check if it's this player's turn.
		if evt.PlayerName != current {
			_ = srv.SendToPlayer(evt.PlayerName, game.Message{
				Type:    game.MsgError,
				Payload: "还没轮到你投票",
			})
			continue
		}

		// Build alive player list for validation.
		aliveNames := make([]string, 0, len(players))
		for _, p := range players {
			if p.Alive {
				aliveNames = append(aliveNames, p.Name)
			}
		}

		// Try to record the vote.
		err := round.RecordVote(evt.PlayerName, evt.Target, aliveNames)
		if err != nil {
			_ = srv.SendToPlayer(evt.PlayerName, game.Message{
				Type:    game.MsgError,
				Payload: err.Error(),
			})
			continue
		}

		// Valid vote: send TURN to the next voter, if any.
		if !round.AllDone() {
			srv.BroadcastToNamedPlayers(game.Message{
				Type:    game.MsgTurn,
				Payload: round.CurrentVoter(),
			})
		}
	}

	// Tally votes and broadcast RESULT.
	tally := round.Tally()
	resultParts := make([]string, 0, len(tally))
	for name, count := range tally {
		resultParts = append(resultParts, fmt.Sprintf("%s:%d", name, count))
	}
	resultPayload := strings.Join(resultParts, ",")
	srv.BroadcastToNamedPlayers(game.Message{
		Type:    game.MsgResult,
		Payload: resultPayload,
	})

	// Find and mark eliminated player.
	eliminated, _ := round.FindEliminated()
	if eliminated != "" {
		for _, p := range players {
			if p.Name == eliminated {
				p.Alive = false
				break
			}
		}
	}

	return &voteResult{Round: round, Eliminated: eliminated}
}

// broadcastReady sends the READY message to all named players.
func broadcastReady(disp *client.Display, srv *server.Server) {
	disp.Info("0000", "游戏开始！")
	srv.BroadcastToNamedPlayers(game.Message{Type: game.MsgReady, Payload: ""})
}

// sendRoleToPlayers privately sends a ROLE message to each player.
// The message format is ROLE|RoleName|word, e.g. ROLE|Civilian|苹果.
// Blank players receive "你是白板" as the word.
func sendRoleToPlayers(disp *client.Display, srv *server.Server, players []*game.Player) {
	for _, p := range players {
		var word string
		var roleName string
		switch p.Role {
		case game.Civilian:
			roleName = "Civilian"
			word = p.Word
		case game.Undercover:
			roleName = "Undercover"
			word = p.Word
		case game.Blank:
			roleName = "Blank"
			word = "你是白板"
		}
		payload := roleName + "|" + word
		msg := game.Message{Type: game.MsgRole, Payload: payload}
		if err := srv.SendToPlayer(p.Name, msg); err != nil {
			disp.Warn(fmt.Sprintf("发送 ROLE 给 %s 失败: %v", p.Name, err))
		}
	}
}
