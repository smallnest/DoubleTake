package game

import (
	"errors"
	"fmt"
	"strings"
)

// ErrVoteNotTied indicates a player tried to vote for someone not in the tied set.
var ErrVoteNotTied = errors.New("can only vote for tied players")

// PKRound manages a single PK round triggered by a voting tie.
// Tied players describe again, then all alive players vote among the tied players.
type PKRound struct {
	Num         int               // PK round number (1-based)
	Tied        []string          // tied player names
	TiedSet     map[string]bool   // fast lookup for tied players
	Phase       string            // "desc" or "vote"
	Desc        *DescRound        // description phase for tied players
	Votes       map[string]string // voter → target
	VoterOrder  []string          // all alive players in voting order
	CurrentVote int               // index of current voter
}

// NewPKRound creates a PK round for the given tied players and all alive players.
func NewPKRound(pkNum int, tied []string, alivePlayers []string) (*PKRound, error) {
	if len(tied) < 2 {
		return nil, fmt.Errorf("PK requires at least 2 tied players, got %d", len(tied))
	}

	tiedSet := make(map[string]bool, len(tied))
	for _, name := range tied {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("tied player name must not be empty")
		}
		if tiedSet[name] {
			return nil, fmt.Errorf("duplicate tied player name: %s", name)
		}
		tiedSet[name] = true
	}

	// Validate all tied players are in alivePlayers.
	aliveSet := make(map[string]bool, len(alivePlayers))
	for _, name := range alivePlayers {
		aliveSet[name] = true
	}
	for _, name := range tied {
		if !aliveSet[name] {
			return nil, fmt.Errorf("tied player %s is not alive", name)
		}
	}

	// Create a description round for tied players only.
	// NewDescRound validates no empty names or duplicates, which we already checked.
	desc, _ := NewDescRound(pkNum, tied)

	// Voter order: all alive players.
	voters := make([]string, len(alivePlayers))
	copy(voters, alivePlayers)

	return &PKRound{
		Num:         pkNum,
		Tied:        tied,
		TiedSet:     tiedSet,
		Phase:       "desc",
		Desc:        desc,
		Votes:       make(map[string]string),
		VoterOrder:  voters,
		CurrentVote: 0,
	}, nil
}

// CurrentSpeaker returns the current PK speaker, or empty if desc phase is done.
func (p *PKRound) CurrentSpeaker() string {
	return p.Desc.CurrentSpeaker()
}

// RecordDesc records a description from a tied player during PK.
func (p *PKRound) RecordDesc(playerName, desc string) error {
	return p.Desc.RecordDesc(playerName, desc)
}

// StartVote switches the PK round to the voting phase.
func (p *PKRound) StartVote() {
	p.Phase = "vote"
}

// CurrentVoter returns the name of the player who should vote now.
func (p *PKRound) CurrentVoter() string {
	if p.CurrentVote >= len(p.VoterOrder) {
		return ""
	}
	return p.VoterOrder[p.CurrentVote]
}

// RecordVote records a PK vote. Only votes for tied players are valid.
func (p *PKRound) RecordVote(voter, target string, alivePlayers []string) error {
	if strings.TrimSpace(target) == "" {
		return ErrVoteEmpty
	}
	if p.CurrentVote >= len(p.VoterOrder) || p.VoterOrder[p.CurrentVote] != voter {
		return ErrNotYourTurn
	}
	if voter == target {
		return ErrVoteSelf
	}

	// Check target is alive.
	aliveSet := make(map[string]bool, len(alivePlayers))
	for _, name := range alivePlayers {
		aliveSet[name] = true
	}
	if !aliveSet[target] {
		return ErrVoteEliminated
	}

	// Check target is one of the tied players.
	if !p.TiedSet[target] {
		return ErrVoteNotTied
	}

	p.Votes[voter] = target
	p.CurrentVote++
	return nil
}

// AllVoted returns true when all alive players have voted.
func (p *PKRound) AllVoted() bool {
	return p.CurrentVote >= len(p.VoterOrder)
}

// Tally returns a map of player name → number of votes received.
func (p *PKRound) Tally() map[string]int {
	result := make(map[string]int)
	for _, target := range p.Votes {
		result[target]++
	}
	return result
}

// FindEliminated returns the tied player with the most votes.
// If there is still a tie, returns empty string and tie=true.
func (p *PKRound) FindEliminated() (string, bool) {
	tally := p.Tally()
	if len(tally) == 0 {
		return "", false
	}

	var maxVotes int
	var topPlayer string
	tie := false

	// Only consider tied players.
	for _, name := range p.Tied {
		count := tally[name]
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
