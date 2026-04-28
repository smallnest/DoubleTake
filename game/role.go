package game

import (
	"errors"
	"fmt"
	"math/rand"
)

// Role represents a player's role in the game.
type Role int

const (
	// Civilian is a normal player who knows the real word.
	Civilian Role = iota
	// Undercover is a player who knows a similar but different word.
	Undercover
	// Blank is a player who knows no word at all.
	Blank
)

// String implements fmt.Stringer for Role.
func (r Role) String() string {
	switch r {
	case Civilian:
		return "Civilian"
	case Undercover:
		return "Undercover"
	case Blank:
		return "Blank"
	default:
		return "Unknown"
	}
}

// Player represents a participant in the game.
type Player struct {
	Name      string
	Role      Role
	Word      string
	Alive     bool
	Connected bool
}

// AssignRoles creates a shuffled list of players with randomly assigned roles.
// It assigns numUndercover Undercover roles and numBlank Blank roles;
// all remaining players are assigned the Civilian role.
// Word is left empty for all players; it should be set in a later stage.
func AssignRoles(names []string, numUndercover, numBlank int) ([]*Player, error) {
	if len(names) == 0 {
		return nil, errors.New("names list must not be empty")
	}
	for _, name := range names {
		if name == "" {
			return nil, errors.New("player name must not be empty")
		}
	}
	if numUndercover < 0 {
		return nil, errors.New("numUndercover must not be negative")
	}
	if numBlank < 0 {
		return nil, errors.New("numBlank must not be negative")
	}
	numCivilian := len(names) - numUndercover - numBlank
	if numUndercover+numBlank >= numCivilian {
		return nil, fmt.Errorf("undercover (%d) + blank (%d) must be less than civilian count (%d)", numUndercover, numBlank, numCivilian)
	}

	// Build index slice and shuffle it.
	indices := make([]int, len(names))
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	players := make([]*Player, len(names))
	for i, idx := range indices {
		var role Role
		switch {
		case i < numUndercover:
			role = Undercover
		case i < numUndercover+numBlank:
			role = Blank
		default:
			role = Civilian
		}
		players[i] = &Player{
			Name:      names[idx],
			Role:      role,
			Alive:     true,
			Connected: false,
		}
	}
	return players, nil
}
