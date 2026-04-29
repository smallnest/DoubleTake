package game

import (
	"errors"
	"fmt"
	"strings"
)

// ErrEmptyDesc indicates the submitted description is blank (whitespace only).
var ErrEmptyDesc = errors.New("description must not be empty")

// ErrNotYourTurn indicates a player tried to describe when it's not their turn.
var ErrNotYourTurn = errors.New("it is not your turn")

// ErrVoteEmpty indicates the vote target is empty or whitespace only.
var ErrVoteEmpty = errors.New("vote target must not be empty")

// ErrVoteNotAPlayer indicates the voter is not in the player list.
var ErrVoteNotAPlayer = errors.New("voter is not a player")

// ErrVoteTargetNotAPlayer indicates the vote target is not in the player list.
var ErrVoteTargetNotAPlayer = errors.New("vote target is not a player")

// ErrVoteAlreadyVoted indicates the voter has already voted.
var ErrVoteAlreadyVoted = errors.New("voter has already voted")

// DescRound manages the description phase of a single round.
// Players speak in the order given by SpeakerOrder; the judge records
// each description for later review.
type DescRound struct {
	RoundNum      int               // round number (1-based)
	SpeakerOrder  []string          // alive players in speaking order
	CurrentIndex  int               // index of the current speaker in SpeakerOrder
	Descriptions  map[string]string // player name → their description
}

// NewDescRound creates a description round for the given alive players.
// It returns an error if the list contains empty names or duplicates.
func NewDescRound(roundNum int, alivePlayers []string) (*DescRound, error) {
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
	order := make([]string, len(alivePlayers))
	copy(order, alivePlayers)
	return &DescRound{
		RoundNum:     roundNum,
		SpeakerOrder: order,
		CurrentIndex: 0,
		Descriptions: make(map[string]string),
	}, nil
}

// CurrentSpeaker returns the name of the player who should speak now.
// Returns empty string if all players have spoken.
func (d *DescRound) CurrentSpeaker() string {
	if d.CurrentIndex >= len(d.SpeakerOrder) {
		return ""
	}
	return d.SpeakerOrder[d.CurrentIndex]
}

// RecordDesc records a player's description and advances the turn index.
// It returns ErrEmptyDesc if the description is blank, ErrNotYourTurn if
// the player is not the current speaker.
func (d *DescRound) RecordDesc(playerName, desc string) error {
	if strings.TrimSpace(desc) == "" {
		return ErrEmptyDesc
	}
	if d.CurrentIndex >= len(d.SpeakerOrder) || d.SpeakerOrder[d.CurrentIndex] != playerName {
		return ErrNotYourTurn
	}
	d.Descriptions[playerName] = desc
	d.CurrentIndex++
	return nil
}

// AllDone returns true when every player in the speaker order has described.
func (d *DescRound) AllDone() bool {
	return d.CurrentIndex >= len(d.SpeakerOrder)
}

// Description returns the description submitted by the given player.
// If the player has not yet described, the second return value is false.
func (d *DescRound) Description(player string) (string, bool) {
	v, ok := d.Descriptions[player]
	return v, ok
}

// VoteRound manages a single round of voting (non-PK).
// All alive players both vote and can be voted for.
type VoteRound struct {
	RoundNum   int
	Players    []string          // all alive players (voters and targets)
	Votes      map[string]string // voter → target
}

// NewVoteRound creates a vote round for the given alive players.
func NewVoteRound(roundNum int, players []string) (*VoteRound, error) {
	if len(players) == 0 {
		return nil, errors.New("players list must not be empty")
	}
	seen := make(map[string]bool, len(players))
	for _, name := range players {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("player name must not be empty")
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate player name: %s", name)
		}
		seen[name] = true
	}
	order := make([]string, len(players))
	copy(order, players)
	return &VoteRound{
		RoundNum: roundNum,
		Players:  order,
		Votes:    make(map[string]string),
	}, nil
}

// CurrentVoter returns the name of the first player who hasn't voted yet.
func (vr *VoteRound) CurrentVoter() string {
	for _, name := range vr.Players {
		if _, voted := vr.Votes[name]; !voted {
			return name
		}
	}
	return ""
}

// RecordVote records a vote from a voter for a target player.
func (vr *VoteRound) RecordVote(voter, target string) error {
	if strings.TrimSpace(target) == "" {
		return ErrVoteEmpty
	}
	valid := false
	for _, name := range vr.Players {
		if name == voter {
			valid = true
			break
		}
	}
	if !valid {
		return ErrVoteNotAPlayer
	}
	if _, ok := vr.Votes[voter]; ok {
		return ErrVoteAlreadyVoted
	}
	validTarget := false
	for _, name := range vr.Players {
		if name == target {
			validTarget = true
			break
		}
	}
	if !validTarget {
		return ErrVoteTargetNotAPlayer
	}
	vr.Votes[voter] = target
	return nil
}

// AllVoted returns true when all players have voted.
func (vr *VoteRound) AllVoted() bool {
	return len(vr.Votes) == len(vr.Players)
}

// Tally returns the vote count for each player.
func (vr *VoteRound) Tally() map[string]int {
	result := make(map[string]int, len(vr.Players))
	for _, name := range vr.Players {
		result[name] = 0
	}
	for _, target := range vr.Votes {
		result[target]++
	}
	return result
}

// FindEliminated returns the player with the most votes and whether there is a tie.
func (vr *VoteRound) FindEliminated() (player string, tie bool) {
	tally := vr.Tally()
	var maxVotes int
	var topPlayers []string
	for _, name := range vr.Players {
		v := tally[name]
		if v > maxVotes {
			maxVotes = v
			topPlayers = []string{name}
		} else if v == maxVotes && v > 0 {
			topPlayers = append(topPlayers, name)
		}
	}
	if len(topPlayers) == 1 {
		return topPlayers[0], false
	}
	return "", true
}
