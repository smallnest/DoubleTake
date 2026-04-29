package game

import (
	"errors"
	"fmt"
	"strings"
)

// ErrPKVoteTargetInvalid indicates the voted target is not in the tied players list.
var ErrPKVoteTargetInvalid = errors.New("vote target must be one of the tied players")

// ErrPKVoteEmpty indicates the vote target is empty or whitespace only.
var ErrPKVoteEmpty = errors.New("vote target must not be empty")

// ErrPKAlreadyVoted indicates the voter has already cast a vote in this PK round.
var ErrPKAlreadyVoted = errors.New("voter has already voted in this PK round")

// ErrPKNotAliveVoter indicates the voter is not in the alive players list.
var ErrPKNotAliveVoter = errors.New("voter is not an alive player")

// PKRound manages a single PK (tie-break) round.
// Only tied players describe; all alive players vote (but only for tied players).
type PKRound struct {
	PKNum       int               // PK round number (1-based)
	TiedPlayers []string          // players who tied in the previous vote
	Desc        *DescRound        // embedded description phase for tied players
	Votes       map[string]string // voter name → target name
	VoterOrder  []string          // alive players who can vote
}

// NewPKRound creates a PK round for the given tied players.
// tiedPlayers must be non-empty with no duplicates and a subset of alivePlayers.
// alivePlayers must be non-empty with no duplicates.
func NewPKRound(pkNum int, tiedPlayers, alivePlayers []string) (*PKRound, error) {
	if len(tiedPlayers) == 0 {
		return nil, errors.New("tied players list must not be empty")
	}
	if len(alivePlayers) == 0 {
		return nil, errors.New("alive players list must not be empty")
	}

	// Validate tied players: no empty names, no duplicates.
	seen := make(map[string]bool, len(tiedPlayers))
	for _, name := range tiedPlayers {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("tied player name must not be empty")
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate tied player name: %s", name)
		}
		seen[name] = true
	}

	// Validate alive players: no empty names, no duplicates.
	seenAlive := make(map[string]bool, len(alivePlayers))
	for _, name := range alivePlayers {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("alive player name must not be empty")
		}
		if seenAlive[name] {
			return nil, fmt.Errorf("duplicate alive player name: %s", name)
		}
		seenAlive[name] = true
	}

	// Every tied player must also be an alive player.
	for _, name := range tiedPlayers {
		if !seenAlive[name] {
			return nil, fmt.Errorf("tied player %q is not in alive players list", name)
		}
	}

	// Create DescRound for tied players only, using pkNum as the round number.
	desc, err := NewDescRound(pkNum, tiedPlayers)
	if err != nil {
		return nil, fmt.Errorf("create desc round for PK: %w", err)
	}

	// Copy slices to avoid shared backing array.
	tiedCopy := make([]string, len(tiedPlayers))
	copy(tiedCopy, tiedPlayers)
	voterCopy := make([]string, len(alivePlayers))
	copy(voterCopy, alivePlayers)

	return &PKRound{
		PKNum:       pkNum,
		TiedPlayers: tiedCopy,
		Desc:        desc,
		Votes:       make(map[string]string),
		VoterOrder:  voterCopy,
	}, nil
}

// CurrentSpeaker returns the name of the tied player who should speak now.
// Delegates to the embedded DescRound.
func (pk *PKRound) CurrentSpeaker() string {
	return pk.Desc.CurrentSpeaker()
}

// RecordPKDesc records a description from a tied player.
// Delegates to the embedded DescRound.
func (pk *PKRound) RecordPKDesc(playerName, desc string) error {
	return pk.Desc.RecordDesc(playerName, desc)
}

// DescAllDone returns true when all tied players have described.
func (pk *PKRound) DescAllDone() bool {
	return pk.Desc.AllDone()
}

// Description returns the description submitted by the given player.
func (pk *PKRound) Description(player string) (string, bool) {
	return pk.Desc.Description(player)
}

// CurrentVoter returns the first alive player who hasn't voted yet.
// Returns empty string if all alive players have voted.
func (pk *PKRound) CurrentVoter() string {
	for _, v := range pk.VoterOrder {
		if _, voted := pk.Votes[v]; !voted {
			return v
		}
	}
	return ""
}

// RecordPKVote records a vote from a voter for a target player.
// The voter must be in the alive players list and must not have voted yet.
// The target must be one of the tied players.
func (pk *PKRound) RecordPKVote(voter, target string) error {
	// Validate target is not empty.
	if strings.TrimSpace(target) == "" {
		return ErrPKVoteEmpty
	}

	// Validate voter is an alive player.
	alive := false
	for _, name := range pk.VoterOrder {
		if name == voter {
			alive = true
			break
		}
	}
	if !alive {
		return ErrPKNotAliveVoter
	}

	// Check if voter already voted.
	if _, ok := pk.Votes[voter]; ok {
		return ErrPKAlreadyVoted
	}

	// Validate target is a tied player.
	tied := false
	for _, name := range pk.TiedPlayers {
		if name == target {
			tied = true
			break
		}
	}
	if !tied {
		return ErrPKVoteTargetInvalid
	}

	pk.Votes[voter] = target
	return nil
}

// AllVotesDone returns true when all alive players have voted.
func (pk *PKRound) AllVotesDone() bool {
	return len(pk.Votes) == len(pk.VoterOrder)
}

// Tally returns the vote count for each tied player.
func (pk *PKRound) Tally() map[string]int {
	result := make(map[string]int, len(pk.TiedPlayers))
	for _, name := range pk.TiedPlayers {
		result[name] = 0
	}
	for _, target := range pk.Votes {
		result[target]++
	}
	return result
}

// FindEliminated returns the player with the most votes and whether there is a tie.
// If there is a unique highest vote getter, returns (name, false).
// If tied (two or more players share the top count, or no votes cast), returns ("", true).
func (pk *PKRound) FindEliminated() (player string, tie bool) {
	tally := pk.Tally()

	var maxVotes int
	var topPlayers []string

	for _, name := range pk.TiedPlayers {
		v := tally[name]
		if v > maxVotes {
			maxVotes = v
			topPlayers = []string{name}
		} else if v == maxVotes && v > 0 {
			topPlayers = append(topPlayers, name)
		}
	}

	// topPlayers is empty when no votes were cast (all zeros); treat as tie.
	if len(topPlayers) == 1 {
		return topPlayers[0], false
	}
	return "", true
}
