package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

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

// guessResult represents the outcome of a guess attempt.
type guessResult int

const (
	guessIgnored guessResult = iota // not a valid guess (not blank, already guessed, etc.)
	guessCorrect                    // guess matches civilian word
	guessWrong                      // guess does not match civilian word
)

// disconnectTimeout is the time to wait for a disconnected player to respond
// before automatically skipping their turn. Override in tests for faster execution.
var disconnectTimeout = 60 * time.Second

// stopTimer safely stops a timer and drains its channel.
func stopTimer(t *time.Timer) {
	if t != nil {
		t.Stop()
		select {
		case <-t.C:
		default:
		}
	}
}

// isPlayerDisconnected checks whether the named player is currently disconnected.
func isPlayerDisconnected(name string, players []*game.Player, playersMu *sync.Mutex) bool {
	playersMu.Lock()
	defer playersMu.Unlock()
	for _, p := range players {
		if p.Name == name {
			return !p.Connected
		}
	}
	return false
}

// checkGuess validates a guess attempt from a player.
// It must be called with playersMu held.
func checkGuess(evt server.GuessEvent, players []*game.Player, guessedThisRound map[string]bool, civilianWord string) guessResult {
	// Find the player and validate they are alive and Blank.
	var player *game.Player
	for _, p := range players {
		if p.Name == evt.PlayerName {
			player = p
			break
		}
	}
	if player == nil || !player.Alive || player.Role != game.Blank {
		return guessIgnored
	}

	// Check if already guessed this round.
	if guessedThisRound[player.Name] {
		return guessIgnored
	}
	guessedThisRound[player.Name] = true

	// Compare guess with civilian word (case-insensitive, trim spaces).
	guess := strings.TrimSpace(evt.Word)
	expected := strings.TrimSpace(civilianWord)
	if strings.EqualFold(guess, expected) {
		return guessCorrect
	}
	return guessWrong
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

	localIP, err := game.GetLocalIP()
	if err != nil {
		localIP = "127.0.0.1"
	}
	roomCode := game.EncodeRoomCode(localIP + ":" + port)
	disp.Info("0000", fmt.Sprintf("房间已创建，房间码: %s (等待 %d 名玩家加入...)", roomCode, cfg.TotalPlayers))

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
	srv.SetGamePlayers(players)
	sendRoleToPlayers(disp, srv, players)

	broadcastReady(disp, srv)

	var playersMu sync.Mutex

	// Start reconnect handler goroutine.
	var currentRound int
	stopReconnect := make(chan struct{})
	go func() {
		for {
			select {
			case req := <-srv.OnReconnect:
				playersMu.Lock()
				var word string
				var aliveList []string
				for _, p := range players {
					if p.Name == req.PlayerName {
						if p.Role == game.Blank {
							word = "你是白板"
						} else {
							word = p.Word
						}
					}
					if p.Alive {
						aliveList = append(aliveList, p.Name)
					}
				}
				round := currentRound
				playersMu.Unlock()
				req.Response <- server.ReconnectResponse{
					Round:        round,
					Word:         word,
					AlivePlayers: aliveList,
				}
			case <-stopReconnect:
				return
			}
		}
	}()

	// Start guess handler goroutine.
	guessedThisRound := make(map[string]bool)
	guessDone := make(chan struct{})
	go func() {
		for {
			select {
			case evt := <-srv.OnGuessMsg:
				playersMu.Lock()
				result := checkGuess(evt, players, guessedThisRound, civilianWord)
				playersMu.Unlock()
				switch result {
				case guessCorrect:
					winPayload := buildWinPayload(game.Blank, players, civilianWord, undercoverWord)
					srv.BroadcastToNamedPlayers(game.Message{Type: game.MsgWin, Payload: winPayload})
					disp.Info("0000", "白板猜对了！白板获胜")
				case guessWrong:
					_ = srv.SendToPlayer(evt.PlayerName, game.Message{
						Type:    game.MsgError,
						Payload: "猜测错误",
					})
				case guessIgnored:
					// Do nothing.
				}
			case <-guessDone:
				return
			}
		}
	}()

	// Game loop: repeat description + voting until game ends.
	alivePlayers := getAlivePlayers(players)
	for roundNum := 1; ; roundNum++ {
		// Update shared round number for reconnect handler.
		playersMu.Lock()
		currentRound = roundNum
		guessedThisRound = make(map[string]bool)
		playersMu.Unlock()

		// Description phase.
		descResult := descriptionPhase(disp, srv, roundNum, alivePlayers, players, &playersMu)
		if descResult == nil {
			close(stopReconnect)
			close(guessDone)
			return cfg
		}

		// Voting phase.
		voteResult := votingPhase(disp, srv, roundNum, alivePlayers, players, &playersMu)
		if voteResult == nil {
			close(stopReconnect)
			close(guessDone)
			return cfg
		}

		// Check win condition after elimination.
		winner, gameOver := game.CheckWinCondition(players)
		if gameOver {
			winPayload := buildWinPayload(winner, players, civilianWord, undercoverWord)
			srv.BroadcastToNamedPlayers(game.Message{Type: game.MsgWin, Payload: winPayload})
			disp.Info("0000", fmt.Sprintf("游戏结束！%s 获胜", winner))
			close(stopReconnect)
			close(guessDone)
			return cfg
		}

		// Update alive players for next round.
		alivePlayers = getAlivePlayers(players)
	}
}

// getAlivePlayers returns the names of all alive players.
func getAlivePlayers(players []*game.Player) []string {
	var alive []string
	for _, p := range players {
		if p.Alive {
			alive = append(alive, p.Name)
		}
	}
	return alive
}

// buildWinPayload constructs the WIN message payload.
// Format: "winner|player1:Role:alive,player2:Role:alive,...|civilianWord|undercoverWord"
func buildWinPayload(winner game.Role, players []*game.Player, civilianWord, undercoverWord string) string {
	var playerStates []string
	for _, p := range players {
		alive := "0"
		if p.Alive {
			alive = "1"
		}
		playerStates = append(playerStates, fmt.Sprintf("%s:%s:%s", p.Name, p.Role, alive))
	}
	return fmt.Sprintf("%s|%s|%s|%s", winner, strings.Join(playerStates, ","), civilianWord, undercoverWord)
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
func descriptionPhase(disp *client.Display, srv *server.Server, roundNum int, alivePlayers []string, players []*game.Player, playersMu *sync.Mutex) *descResult {
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
		current := round.CurrentSpeaker()
		if current == "" {
			break
		}

		var timer *time.Timer
		var timeoutCh <-chan time.Time
		if isPlayerDisconnected(current, players, playersMu) {
			timer = time.NewTimer(disconnectTimeout)
			timeoutCh = timer.C
		}

		select {
		case evt := <-srv.OnDescMsg:
			stopTimer(timer)
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

		case disc := <-srv.OnDisconnect:
			if disc.PlayerName == current {
				stopTimer(timer)
				timer = time.NewTimer(disconnectTimeout)
				timeoutCh = timer.C
			}
			continue

		case <-timeoutCh:
			disp.Warn(fmt.Sprintf("%s 超时未发言（已掉线），跳过", current))
			round.SkipCurrent()
			if !round.AllDone() {
				srv.BroadcastToNamedPlayers(game.Message{
					Type:    game.MsgTurn,
					Payload: round.CurrentSpeaker(),
				})
			}
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
// If a tie is detected, it enters the PK phase (pkPhase) to resolve the tie.
// It returns the eliminated player name (empty if unresolved) or nil on failure.
func votingPhase(disp *client.Display, srv *server.Server, roundNum int, alivePlayers []string, players []*game.Player, playersMu *sync.Mutex) *voteResult {
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
		current := round.CurrentVoter()
		if current == "" {
			break
		}

		var timer *time.Timer
		var timeoutCh <-chan time.Time
		if isPlayerDisconnected(current, players, playersMu) {
			timer = time.NewTimer(disconnectTimeout)
			timeoutCh = timer.C
		}

		select {
		case evt := <-srv.OnVoteMsg:
			stopTimer(timer)
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

		case disc := <-srv.OnDisconnect:
			if disc.PlayerName == current {
				stopTimer(timer)
				timer = time.NewTimer(disconnectTimeout)
				timeoutCh = timer.C
			}
			continue

		case <-timeoutCh:
			disp.Warn(fmt.Sprintf("%s 超时未投票（已掉线），跳过", current))
			round.SkipCurrent()
			if !round.AllDone() {
				srv.BroadcastToNamedPlayers(game.Message{
					Type:    game.MsgTurn,
					Payload: round.CurrentVoter(),
				})
			}
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
		playersMu.Lock()
		for _, p := range players {
			if p.Name == eliminated {
				p.Alive = false
				break
			}
		}
		playersMu.Unlock()
		return &voteResult{Round: round, Eliminated: eliminated}
	}

	// Tie detected — find tied players and enter PK phase.
	tied := round.FindTiedPlayers()
	if tied == nil {
		// No votes cast — no elimination.
		return &voteResult{Round: round, Eliminated: ""}
	}

	// Collect current alive players for PK.
	alive := make([]string, 0, len(players))
	for _, p := range players {
		if p.Alive {
			alive = append(alive, p.Name)
		}
	}

	eliminated = pkPhase(disp, srv, tied, alive, players, playersMu)
	return &voteResult{Round: round, Eliminated: eliminated}
}

// pkPhase runs the PK phase to resolve a voting tie.
// It loops PK rounds until a single player is eliminated.
// Returns the name of the eliminated player, or empty if unresolved.
func pkPhase(disp *client.Display, srv *server.Server, tied []string, alivePlayers []string, players []*game.Player, playersMu *sync.Mutex) string {
	pkNum := 1
	for {
		disp.Info("0000", fmt.Sprintf("平票！进入 PK 第 %d 轮: %s", pkNum, strings.Join(tied, ", ")))

		// Broadcast PK_START|pkNum|tiedPlayerList.
		tiedList := strings.Join(tied, ",")
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgPKStart,
			Payload: fmt.Sprintf("%d|%s", pkNum, tiedList),
		})

		pk, err := game.NewPKRound(pkNum, tied, alivePlayers)
		if err != nil {
			disp.Warn(fmt.Sprintf("创建 PK 轮次失败: %v", err))
			return ""
		}

		// --- PK Description phase: only tied players describe ---
		// Send TURN to the first tied speaker.
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgTurn,
			Payload: pk.CurrentSpeaker(),
		})

		for !pk.Desc.AllDone() {
			current := pk.CurrentSpeaker()
			if current == "" {
				break
			}

			var timer *time.Timer
			var timeoutCh <-chan time.Time
			if isPlayerDisconnected(current, players, playersMu) {
				timer = time.NewTimer(disconnectTimeout)
				timeoutCh = timer.C
			}

			select {
			case evt := <-srv.OnDescMsg:
				stopTimer(timer)
				if evt.PlayerName != current {
					_ = srv.SendToPlayer(evt.PlayerName, game.Message{
						Type:    game.MsgError,
						Payload: "还没轮到你发言",
					})
					continue
				}

				err := pk.RecordDesc(evt.PlayerName, evt.Description)
				if err == game.ErrEmptyDesc {
					_ = srv.SendToPlayer(evt.PlayerName, game.Message{
						Type:    game.MsgError,
						Payload: "描述不能为空，请重新输入",
					})
					continue
				}
				if err != nil {
					_ = srv.SendToPlayer(evt.PlayerName, game.Message{
						Type:    game.MsgError,
						Payload: err.Error(),
					})
					continue
				}

				// Broadcast DESC|playerName|description.
				srv.BroadcastToNamedPlayers(game.Message{
					Type:    game.MsgDesc,
					Payload: evt.PlayerName + "|" + evt.Description,
				})

				if !pk.Desc.AllDone() {
					srv.BroadcastToNamedPlayers(game.Message{
						Type:    game.MsgTurn,
						Payload: pk.CurrentSpeaker(),
					})
				}

			case disc := <-srv.OnDisconnect:
				if disc.PlayerName == current {
					stopTimer(timer)
					timer = time.NewTimer(disconnectTimeout)
					timeoutCh = timer.C
				}
				continue

			case <-timeoutCh:
				disp.Warn(fmt.Sprintf("PK %s 超时未发言（已掉线），跳过", current))
				pk.Desc.SkipCurrent()
				if !pk.Desc.AllDone() {
					srv.BroadcastToNamedPlayers(game.Message{
						Type:    game.MsgTurn,
						Payload: pk.CurrentSpeaker(),
					})
				}
			}
		}

		// --- PK Vote phase: all alive players vote among tied ---
		pk.StartVote()

		// Refresh alive players list.
		alivePlayers = nil
		for _, p := range players {
			if p.Alive {
				alivePlayers = append(alivePlayers, p.Name)
			}
		}

		// Broadcast PK_VOTE|pkNum|tiedPlayerList.
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgPKVote,
			Payload: fmt.Sprintf("%d|%s", pkNum, tiedList),
		})

		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgTurn,
			Payload: pk.CurrentVoter(),
		})

		for !pk.AllVoted() {
			current := pk.CurrentVoter()
			if current == "" {
				break
			}

			var timer *time.Timer
			var timeoutCh <-chan time.Time
			if isPlayerDisconnected(current, players, playersMu) {
				timer = time.NewTimer(disconnectTimeout)
				timeoutCh = timer.C
			}

			select {
			case evt := <-srv.OnVoteMsg:
				stopTimer(timer)
				if evt.PlayerName != current {
					_ = srv.SendToPlayer(evt.PlayerName, game.Message{
						Type:    game.MsgError,
						Payload: "还没轮到你投票",
					})
					continue
				}

				err := pk.RecordVote(evt.PlayerName, evt.Target, alivePlayers)
				if err != nil {
					_ = srv.SendToPlayer(evt.PlayerName, game.Message{
						Type:    game.MsgError,
						Payload: err.Error(),
					})
					continue
				}

				if !pk.AllVoted() {
					srv.BroadcastToNamedPlayers(game.Message{
						Type:    game.MsgTurn,
						Payload: pk.CurrentVoter(),
					})
				}

			case disc := <-srv.OnDisconnect:
				if disc.PlayerName == current {
					stopTimer(timer)
					timer = time.NewTimer(disconnectTimeout)
					timeoutCh = timer.C
				}
				continue

			case <-timeoutCh:
				disp.Warn(fmt.Sprintf("PK %s 超时未投票（已掉线），跳过", current))
				pk.SkipCurrentVoter()
				if !pk.AllVoted() {
					srv.BroadcastToNamedPlayers(game.Message{
						Type:    game.MsgTurn,
						Payload: pk.CurrentVoter(),
					})
				}
			}
		}

		// Tally PK votes and broadcast RESULT.
		tally := pk.Tally()
		resultParts := make([]string, 0, len(tally))
		for name, count := range tally {
			resultParts = append(resultParts, fmt.Sprintf("%s:%d", name, count))
		}
		resultPayload := strings.Join(resultParts, ",")
		srv.BroadcastToNamedPlayers(game.Message{
			Type:    game.MsgResult,
			Payload: resultPayload,
		})

		eliminated, tie := pk.FindEliminated()
		if !tie {
			// Clear winner — mark eliminated.
			if eliminated != "" {
				playersMu.Lock()
				for _, p := range players {
					if p.Name == eliminated {
						p.Alive = false
						break
					}
				}
				playersMu.Unlock()
			}
			return eliminated
		}

		// Still tied — find new tied set and loop.
		pkNum++
		tiedTally := pk.Tally()
		var newTied []string
		var maxVotes int
		for _, name := range tied {
			count := tiedTally[name]
			if count > maxVotes {
				maxVotes = count
				newTied = []string{name}
			} else if count == maxVotes {
				newTied = append(newTied, name)
			}
		}
		tied = newTied
	}
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
