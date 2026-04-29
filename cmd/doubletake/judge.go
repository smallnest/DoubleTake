package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

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
// quitCh is checked to allow the referee to quit during word collection.
func collectWordsFromCh(out io.Writer, disp *client.Display, stdinCh <-chan string, stdinDone <-chan struct{}, quitCh <-chan struct{}) (civilianWord, undercoverWord string) {
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
		case <-quitCh:
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
		case <-quitCh:
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
	quit <-chan struct{} // closed when the user types "quit"
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

	// Set up signal handling for SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	disp.Info("0000", fmt.Sprintf("房间已创建，等待 %d 名玩家加入...", cfg.TotalPlayers))

	// waitingPhase blocks until the referee confirms game start.
	stdinSrc := newStdinSource(scanner)

	// Create a unified quit channel that merges stdin quit, stdin EOF, and signals.
	quitCh := make(chan struct{}, 1)
	go func() {
		select {
		case <-stdinSrc.quit:
		case <-stdinSrc.done:
		case <-sigCh:
		}
		select {
		case quitCh <- struct{}{}:
		default:
		}
	}()

	waitingPhase(out, in, disp, srv, cfg, stdinSrc, quitCh)

	// Check if quit was triggered during waiting phase.
	select {
	case <-quitCh:
		return cfg
	default:
	}

	// Game loop: each iteration is one full game.
	// After a game ends, the referee is prompted "再来一局？(Y/N)".
	var players []*game.Player
	for {
		// Collect words and assign roles.
		civilianWord, undercoverWord := collectWordsFromCh(out, disp, stdinSrc.ch, stdinSrc.done, quitCh)
		if civilianWord == "" && undercoverWord == "" {
			return cfg
		}
		names := srv.PlayerNames()
		var err error
		players, err = game.AssignRoles(names, cfg.Undercovers, cfg.Blanks)
		if err != nil {
			disp.Warn(fmt.Sprintf("角色分配失败: %v", err))
			return cfg
		}
		game.AssignWords(players, civilianWord, undercoverWord)
		sendRoleToPlayers(disp, srv, players)

		broadcastReady(disp, srv)

		// Run game rounds until win condition or quit.
		gameOver := false
		for roundNum := 1; ; roundNum++ {
			aliveNames := collectAliveNames(players)

			// Description phase.
			descResult := descriptionPhase(disp, srv, roundNum, aliveNames, quitCh)
			if descResult == nil {
				return cfg
			}

			// Voting phase.
			eliminated, tie, voteRound := votingPhase(disp, srv, roundNum, players, quitCh)
			if eliminated == "" && !tie {
				// quit triggered during voting
				return cfg
			}

			if eliminated != "" {
				for _, p := range players {
					if p.Name == eliminated {
						p.Alive = false
						break
					}
				}
				disp.Info("0000", fmt.Sprintf("%s 被投票淘汰", eliminated))

				// Broadcast KICK to all players.
				srv.BroadcastToNamedPlayers(game.Message{
					Type:    game.MsgKick,
					Payload: eliminated,
				})
			} else if tie && voteRound != nil {
				disp.Info("0000", "平票，进入 PK 环节")
				tiedPlayers := getTiedPlayers(voteRound)
				pkEliminated := pkPhase(disp, srv, tiedPlayers, players, quitCh)
				if pkEliminated != "" {
					disp.Info("0000", fmt.Sprintf("%s 在 PK 中被淘汰", pkEliminated))
				}
			}

			// Check win condition.
			if winner := game.CheckWinCondition(players); winner != "" {
				disp.Info("0000", fmt.Sprintf("游戏结束：%s 获胜！", winner))
				srv.BroadcastToNamedPlayers(game.Message{
					Type:    game.MsgWin,
					Payload: string(winner),
				})
				gameOver = true
				break
			}
		}

		if !gameOver {
			return cfg
		}

		// "再来一局？(Y/N)" prompt.
		fmt.Fprint(out, "  再来一局？(Y/N): ")
		var answer string
		select {
		case line, ok := <-stdinSrc.ch:
			if !ok {
				return cfg
			}
			answer = strings.TrimSpace(line)
		case <-stdinSrc.done:
			return cfg
		case <-quitCh:
			return cfg
		}

		if strings.EqualFold(answer, "Y") {
			// Broadcast RESTART to all connected players.
			srv.BroadcastToNamedPlayers(game.Message{Type: game.MsgRestart})
			disp.Info("0000", "新一局即将开始...")
			continue // new game
		}

		// N or any other input: broadcast quit and stop.
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgQuit,
			Payload: "裁判结束了游戏",
		})
		return cfg
	}
}

// collectAliveNames returns the names of all alive players.
func collectAliveNames(players []*game.Player) []string {
	var names []string
	for _, p := range players {
		if p.Alive {
			names = append(names, p.Name)
		}
	}
	return names
}

// newStdinSource starts a goroutine that reads from scanner and returns a stdinSource.
// It detects "quit" commands and closes the quit channel accordingly.
func newStdinSource(scanner *bufio.Scanner) *stdinSource {
	ch := make(chan string, 1)
	done := make(chan struct{})
	quit := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.EqualFold(line, "quit") {
				close(quit)
				return
			}
			ch <- line
		}
	}()
	return &stdinSource{ch: ch, done: done, quit: quit}
}

// waitingPhase displays player join events and reads stdin for "start" commands.
// It blocks until the referee confirms game start, stdin is closed, or quit is received.
func waitingPhase(out io.Writer, in io.Reader, disp *client.Display, srv *server.Server, cfg GameConfig, stdinSrc *stdinSource, quitCh <-chan struct{}) {
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
					case <-quitCh:
						return
					}
					continue
				}
				// count == cfg.TotalPlayers (or more, shouldn't happen)
				return
			}

		case <-stdinSrc.done:
			return
		case <-quitCh:
			return
		}
	}
}

// descriptionPhase runs the description phase for one round.
// It broadcasts ROUND|roundNum|speakerOrder, then sends TURN|playerName to the
// first speaker and processes DESC messages until all players have spoken.
// quitCh allows the referee to quit mid-phase.
func descriptionPhase(disp *client.Display, srv *server.Server, roundNum int, alivePlayers []string, quitCh <-chan struct{}) *descResult {
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
		select {
		case evt := <-srv.OnDescMsg:
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
		case <-quitCh:
			return nil
		}
	}

	return &descResult{Round: round}
}

// votingPhase runs a voting round where all alive players vote.
// It broadcasts VOTE|roundNum|playerList, then sends TURN|playerName to
// each voter and processes VOTE messages until all have voted.
// Returns the eliminated player name (empty string if tie), whether a tie occurred,
// and the VoteRound for further analysis (e.g., getting tied players).
// quitCh allows the referee to quit mid-phase.
func votingPhase(disp *client.Display, srv *server.Server, roundNum int, players []*game.Player, quitCh <-chan struct{}) (eliminated string, tie bool, voteRound *game.VoteRound) {
	alivePlayers := make([]string, 0, len(players))
	for _, p := range players {
		if p.Alive {
			alivePlayers = append(alivePlayers, p.Name)
		}
	}

	round, err := game.NewVoteRound(roundNum, alivePlayers)
	if err != nil {
		disp.Warn(fmt.Sprintf("创建投票轮次失败: %v", err))
		return "", false, nil
	}

	// Broadcast VOTE|roundNum|playerList
	playerList := strings.Join(alivePlayers, ",")
	srv.BroadcastToNamedPlayers(game.Message{
		Type:    game.MsgVote,
		Payload: fmt.Sprintf("%d|%s", roundNum, playerList),
	})

	// Send TURN to each voter in order
	for !round.AllVoted() {
		current := round.CurrentVoter()
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgTurn,
			Payload: current,
		})

		var evt server.VoteEvent
		select {
		case evt = <-srv.OnVoteMsg:
		case <-quitCh:
			return "", false, nil
		}

		if evt.PlayerName != current {
			_ = srv.SendToPlayer(evt.PlayerName, game.Message{
				Type:    game.MsgError,
				Payload: "还没轮到你投票",
			})
			continue
		}

		err := round.RecordVote(evt.PlayerName, evt.Target)
		if err != nil {
			_ = srv.SendToPlayer(evt.PlayerName, game.Message{
				Type:    game.MsgError,
				Payload: err.Error(),
			})
			continue
		}

		// Broadcast vote
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgVoteBroadcast,
			Payload: evt.PlayerName + "|" + evt.Target,
		})
	}

	// Broadcast RESULT|tally
	tally := round.Tally()
	var tallyParts []string
	for _, name := range alivePlayers {
		tallyParts = append(tallyParts, fmt.Sprintf("%s:%d", name, tally[name]))
	}
	srv.BroadcastToNamedPlayers(game.Message{
		Type:    game.MsgResult,
		Payload: strings.Join(tallyParts, ","),
	})

	eliminated, tie = round.FindEliminated()
	return eliminated, tie, round
}

// getTiedPlayers extracts the players with the highest vote count from a VoteRound.
func getTiedPlayers(voteRound *game.VoteRound) []string {
	if voteRound == nil {
		return nil
	}
	tally := voteRound.Tally()
	var maxVotes int
	for _, name := range voteRound.Players {
		v := tally[name]
		if v > maxVotes {
			maxVotes = v
		}
	}
	if maxVotes == 0 {
		return nil
	}
	var tied []string
	for _, name := range voteRound.Players {
		if tally[name] == maxVotes {
			tied = append(tied, name)
		}
	}
	return tied
}

// pkPhase runs the PK (tie-break) phase. It loops until a unique eliminated
// player is found. Each PK round:
// 1. Broadcasts PK_START|tiedPlayers
// 2. Runs descriptionPhase for tied players only
// 3. Collects PK votes from all alive players
// 4. Broadcasts RESULT|tally
// 5. Checks for elimination
// quitCh allows the referee to quit mid-phase.
func pkPhase(disp *client.Display, srv *server.Server, tiedPlayers []string, players []*game.Player, quitCh <-chan struct{}) string {
	// Build alive players list
	alivePlayers := make([]string, 0, len(players))
	for _, p := range players {
		if p.Alive {
			alivePlayers = append(alivePlayers, p.Name)
		}
	}

	pkNum := 1
	const maxPKRounds = 3
	currentTied := tiedPlayers

	for pkNum <= maxPKRounds {
		// 1. Broadcast PK_START|tiedPlayers
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgPKStart,
			Payload: strings.Join(currentTied, ","),
		})

		// 2. Run description phase for tied players only
		descRes := descriptionPhase(disp, srv, pkNum, currentTied, quitCh)
		if descRes == nil {
			return ""
		}

		// 3. Collect PK votes from all alive players
		pkRound, err := game.NewPKRound(pkNum, currentTied, alivePlayers)
		if err != nil {
			disp.Warn(fmt.Sprintf("创建 PK 轮次失败: %v", err))
			return ""
		}

		// Broadcast ROUND for PK voting (reuse ROUND message type)
		voterList := strings.Join(alivePlayers, ",")
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgRound,
			Payload: fmt.Sprintf("%d|%s", pkNum, voterList),
		})

		for !pkRound.AllVotesDone() {
			currentVoter := pkRound.CurrentVoter()
			srv.BroadcastToNamedPlayers(game.Message{
				Type:    game.MsgTurn,
				Payload: currentVoter,
			})

			var evt server.VoteEvent
			select {
			case evt = <-srv.OnPKVoteMsg:
			case <-quitCh:
				return ""
			}

			if evt.PlayerName != currentVoter {
				_ = srv.SendToPlayer(evt.PlayerName, game.Message{
					Type:    game.MsgError,
					Payload: "还没轮到你投票",
				})
				continue
			}

			err := pkRound.RecordPKVote(evt.PlayerName, evt.Target)
			if err != nil {
				_ = srv.SendToPlayer(evt.PlayerName, game.Message{
					Type:    game.MsgError,
					Payload: err.Error(),
				})
				continue
			}

			// Broadcast PK vote
			srv.BroadcastToNamedPlayers(game.Message{
				Type:    game.MsgVoteBroadcast,
				Payload: evt.PlayerName + "|" + evt.Target,
			})
		}

		// 4. Broadcast RESULT|tally
		tally := pkRound.Tally()
		var tallyParts []string
		for _, name := range currentTied {
			tallyParts = append(tallyParts, fmt.Sprintf("%s:%d", name, tally[name]))
		}
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgResult,
			Payload: strings.Join(tallyParts, ","),
		})

		// 5. Check for elimination
		eliminated, stillTied := pkRound.FindEliminated()
		if !stillTied {
			// Unique highest vote getter
			for _, p := range players {
				if p.Name == eliminated {
					p.Alive = false
					break
				}
			}
			srv.BroadcastToNamedPlayers(game.Message{
				Type:    game.MsgKick,
				Payload: eliminated,
			})
			return eliminated
		}

		// Still tied: get new tied players from tally and loop
		var maxVotes int
		for _, name := range currentTied {
			v := tally[name]
			if v > maxVotes {
				maxVotes = v
			}
		}
		if maxVotes == 0 {
			// No votes cast, treat as tie among all tied players
			pkNum++
			continue
		}
		var nextTied []string
		for _, name := range currentTied {
			if tally[name] == maxVotes {
				nextTied = append(nextTied, name)
			}
		}
		currentTied = nextTied
		pkNum++
	}

	// Max PK rounds exhausted without resolution
	disp.Warn(fmt.Sprintf("PK %d 轮后仍平票，无人淘汰", maxPKRounds))
	return ""
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
