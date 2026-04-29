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

// CheckWinCondition checks whether the game has ended after an elimination.
// Blank players are counted on the Civilian side.
//   - All Undercover eliminated → (Civilian, true)
//   - Undercover alive >= Civilian+Blank alive → (Undercover, true)
//   - Otherwise → (0, false), game continues
func CheckWinCondition(players []*Player) (winner Role, gameOver bool) {
	var undercoverAlive, civilianAlive int
	for _, p := range players {
		if !p.Alive {
			continue
		}
		if p.Role == Undercover {
			undercoverAlive++
		} else {
			civilianAlive++
		}
	}

	if undercoverAlive == 0 {
		return Civilian, true
	}
	if undercoverAlive >= civilianAlive {
		return Undercover, true
	}
	return 0, false
}
