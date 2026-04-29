package game

import (
	"errors"
	"fmt"
	"strings"
)

// Vote-specific errors.
var (
	// ErrVoteSelf indicates a player tried to vote for themselves.
	ErrVoteSelf = errors.New("cannot vote for yourself")

	// ErrVoteEliminated indicates a player tried to vote for an eliminated player.
	ErrVoteEliminated = errors.New("cannot vote for an eliminated player")

	// ErrVoteUnknown indicates a player tried to vote for someone not in the game.
	ErrVoteUnknown = errors.New("cannot vote for a player not in the game")

	// ErrVoteEmpty indicates a player submitted an empty vote target.
	ErrVoteEmpty = errors.New("vote target must not be empty")
)

// VoteRound manages the voting phase of a single round.
// Players vote one at a time in the order given by Voters; the judge records
// each vote and tallies the results after all votes are in.
type VoteRound struct {
	RoundNum      int               // round number (1-based)
	Voters        []string          // alive players in voting order
	CurrentIndex  int               // index of the current voter in Voters
	Votes         map[string]string // voter name → target name
}

// NewVoteRound creates a voting round for the given alive players.
// It returns an error if the list contains empty names or duplicates.
func NewVoteRound(roundNum int, alivePlayers []string) (*VoteRound, error) {
	seen := make(map[string]bool, len(alivePlayers))
	for _, name := range alivePlayers {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("player name must not be empty")
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate player name: %s", name)
		}
		seen[name] = true
	}
	voters := make([]string, len(alivePlayers))
	copy(voters, alivePlayers)
	return &VoteRound{
		RoundNum:     roundNum,
		Voters:       voters,
		CurrentIndex: 0,
		Votes:        make(map[string]string),
	}, nil
}

// CurrentVoter returns the name of the player who should vote now.
// Returns empty string if all players have voted.
func (v *VoteRound) CurrentVoter() string {
	if v.CurrentIndex >= len(v.Voters) {
		return ""
	}
	return v.Voters[v.CurrentIndex]
}

// RecordVote records a player's vote and advances the turn index.
// alivePlayers is the set of currently alive players used for validation.
// Validation order: empty target → not your turn → vote for self → vote for
// eliminated player → vote for unknown player.
func (v *VoteRound) RecordVote(voter, target string, alivePlayers []string) error {
	if strings.TrimSpace(target) == "" {
		return ErrVoteEmpty
	}
	if v.CurrentIndex >= len(v.Voters) || v.Voters[v.CurrentIndex] != voter {
		return ErrNotYourTurn
	}
	if voter == target {
		return ErrVoteSelf
	}
	aliveSet := make(map[string]bool, len(alivePlayers))
	for _, p := range alivePlayers {
		aliveSet[p] = true
	}
	if !aliveSet[target] {
		// Check if the target is a known voter who was eliminated,
		// or a completely unknown player.
		for _, v := range v.Voters {
			if v == target {
				return ErrVoteEliminated
			}
		}
		return ErrVoteUnknown
	}
	v.Votes[voter] = target
	v.CurrentIndex++
	return nil
}

// SkipCurrent skips the current voter without recording a vote.
// The vote is treated as an abstention. Advances the turn index to the next voter.
func (v *VoteRound) SkipCurrent() {
	if v.CurrentIndex < len(v.Voters) {
		v.CurrentIndex++
	}
}

// AllDone returns true when every player in the voter list has voted.
func (v *VoteRound) AllDone() bool {
	return v.CurrentIndex >= len(v.Voters)
}

// Tally returns a map of player name → number of votes received.
func (v *VoteRound) Tally() map[string]int {
	result := make(map[string]int)
	for _, target := range v.Votes {
		result[target]++
	}
	return result
}

// FindEliminated returns the name of the player with the most votes.
// If there is a tie for the top spot, it returns an empty string and
// the second return value is true.
func (v *VoteRound) FindEliminated() (string, bool) {
	tally := v.Tally()
	if len(tally) == 0 {
		return "", false
	}
	var maxVotes int
	var topPlayer string
	tie := false
	for name, count := range tally {
		if count > maxVotes {
			maxVotes = count
			topPlayer = name
			tie = false
		} else if count == maxVotes {
			tie = true
		}
	}
	if tie {
		return "", true
	}
	return topPlayer, false
}

// FindTiedPlayers returns the names of all players tied for the most votes.
// Returns nil if there are no votes or no tie.
func (v *VoteRound) FindTiedPlayers() []string {
	tally := v.Tally()
	if len(tally) == 0 {
		return nil
	}
	var maxVotes int
	for _, count := range tally {
		if count > maxVotes {
			maxVotes = count
		}
	}
	var tied []string
	for name, count := range tally {
		if count == maxVotes {
			tied = append(tied, name)
		}
	}
	if len(tied) < 2 {
		return nil
	}
	return tied
}
