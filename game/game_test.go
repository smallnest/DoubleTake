package game

import (
	"strings"
	"testing"
)

func TestNewDescRound(t *testing.T) {
	players := []string{"alice", "bob", "carol"}
	d, err := NewDescRound(1, players)
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	if d.RoundNum != 1 {
		t.Errorf("RoundNum = %d, want 1", d.RoundNum)
	}
	if len(d.SpeakerOrder) != 3 {
		t.Fatalf("len(SpeakerOrder) = %d, want 3", len(d.SpeakerOrder))
	}
	if d.CurrentIndex != 0 {
		t.Errorf("CurrentIndex = %d, want 0", d.CurrentIndex)
	}
	if len(d.Descriptions) != 0 {
		t.Errorf("Descriptions should be empty, got %d entries", len(d.Descriptions))
	}
	// Verify the original slice is not shared.
	players[0] = "changed"
	if d.SpeakerOrder[0] == "changed" {
		t.Error("SpeakerOrder shares backing array with input slice")
	}
}

func TestCurrentSpeaker(t *testing.T) {
	d, err := NewDescRound(2, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	if got := d.CurrentSpeaker(); got != "alice" {
		t.Errorf("CurrentSpeaker() = %q, want %q", got, "alice")
	}
}

func TestCurrentSpeaker_AllDone(t *testing.T) {
	d, err := NewDescRound(1, []string{"alice"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}
	d.CurrentIndex = 1

	if got := d.CurrentSpeaker(); got != "" {
		t.Errorf("CurrentSpeaker() = %q, want empty string when all done", got)
	}
}

func TestRecordDesc_NormalFlow(t *testing.T) {
	d, err := NewDescRound(1, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	if err := d.RecordDesc("alice", "it is a fruit"); err != nil {
		t.Fatalf("RecordDesc(alice) error: %v", err)
	}
	if d.CurrentSpeaker() != "bob" {
		t.Errorf("after alice, CurrentSpeaker() = %q, want %q", d.CurrentSpeaker(), "bob")
	}
	if d.Descriptions["alice"] != "it is a fruit" {
		t.Errorf("Descriptions[alice] = %q, want %q", d.Descriptions["alice"], "it is a fruit")
	}

	if err := d.RecordDesc("bob", "you can eat it"); err != nil {
		t.Fatalf("RecordDesc(bob) error: %v", err)
	}
	if d.CurrentSpeaker() != "carol" {
		t.Errorf("after bob, CurrentSpeaker() = %q, want %q", d.CurrentSpeaker(), "carol")
	}

	if err := d.RecordDesc("carol", "red or green"); err != nil {
		t.Fatalf("RecordDesc(carol) error: %v", err)
	}
	if !d.AllDone() {
		t.Error("AllDone() = false, want true after all players described")
	}
	if d.CurrentSpeaker() != "" {
		t.Errorf("CurrentSpeaker() = %q after all done, want empty", d.CurrentSpeaker())
	}
}

func TestRecordDesc_EmptyDescription(t *testing.T) {
	d, err := NewDescRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	tests := []struct {
		name string
		desc string
	}{
		{"empty string", ""},
		{"spaces only", "   "},
		{"tabs only", "\t\t"},
		{"newline only", "\n"},
		{"mixed whitespace", " \t\n "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.RecordDesc("alice", tt.desc)
			if err != ErrEmptyDesc {
				t.Errorf("RecordDesc(%q) err = %v, want ErrEmptyDesc", tt.name, err)
			}
		})
	}
	// Verify index hasn't advanced.
	if d.CurrentIndex != 0 {
		t.Errorf("CurrentIndex = %d after empty desc rejection, want 0", d.CurrentIndex)
	}
}

func TestRecordDesc_NotYourTurn(t *testing.T) {
	d, err := NewDescRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	err = d.RecordDesc("bob", "something")
	if err != ErrNotYourTurn {
		t.Errorf("RecordDesc(bob) err = %v, want ErrNotYourTurn", err)
	}

	// After all done, any player gets ErrNotYourTurn.
	d2, err2 := NewDescRound(1, []string{"alice"})
	if err2 != nil {
		t.Fatalf("NewDescRound error: %v", err2)
	}
	_ = d2.RecordDesc("alice", "ok")
	err = d2.RecordDesc("alice", "another")
	if err != ErrNotYourTurn {
		t.Errorf("RecordDesc after all done: err = %v, want ErrNotYourTurn", err)
	}
}

func TestSkipCurrent(t *testing.T) {
	d, err := NewDescRound(1, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	if d.CurrentSpeaker() != "alice" {
		t.Fatalf("expected alice as current speaker")
	}

	d.SkipCurrent()
	if d.CurrentSpeaker() != "bob" {
		t.Errorf("after skip, expected bob as current speaker, got %q", d.CurrentSpeaker())
	}
	if d.Descriptions["alice"] != "" {
		t.Error("SkipCurrent should not record a description")
	}
}

func TestSkipCurrent_AllSkipped(t *testing.T) {
	d, err := NewDescRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	d.SkipCurrent()
	d.SkipCurrent()
	if !d.AllDone() {
		t.Error("AllDone() should be true after skipping all players")
	}
	if d.CurrentSpeaker() != "" {
		t.Errorf("CurrentSpeaker() should be empty after all skipped")
	}
}

func TestSkipCurrent_NoOpWhenDone(t *testing.T) {
	d, err := NewDescRound(1, []string{"alice"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}
	d.CurrentIndex = 1

	d.SkipCurrent() // should be a no-op
	if d.CurrentIndex != 1 {
		t.Errorf("CurrentIndex should remain 1 after no-op skip, got %d", d.CurrentIndex)
	}
}

func TestAllDone_EmptyPlayers(t *testing.T) {
	d, err := NewDescRound(1, []string{})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}
	if !d.AllDone() {
		t.Error("AllDone() = false for 0 players, want true")
	}
	if got := d.CurrentSpeaker(); got != "" {
		t.Errorf("CurrentSpeaker() = %q for 0 players, want empty", got)
	}
}

func TestAllDone_OnePlayer(t *testing.T) {
	d, err := NewDescRound(1, []string{"solo"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}
	if d.AllDone() {
		t.Error("AllDone() = true before solo player describes")
	}
	if err := d.RecordDesc("solo", "hello"); err != nil {
		t.Fatalf("RecordDesc(solo) error: %v", err)
	}
	if !d.AllDone() {
		t.Error("AllDone() = false after solo player describes")
	}
}

func TestDescRound_RecordsAllDescriptions(t *testing.T) {
	players := []string{"alice", "bob", "carol"}
	d, err := NewDescRound(3, players)
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	descs := map[string]string{
		"alice": "round and sweet",
		"bob":   "grows on trees",
		"carol": "red or green",
	}
	for _, p := range players {
		if err := d.RecordDesc(p, descs[p]); err != nil {
			t.Fatalf("RecordDesc(%s) error: %v", p, err)
		}
	}

	if len(d.Descriptions) != 3 {
		t.Errorf("len(Descriptions) = %d, want 3", len(d.Descriptions))
	}
	for _, p := range players {
		if d.Descriptions[p] != descs[p] {
			t.Errorf("Descriptions[%s] = %q, want %q", p, d.Descriptions[p], descs[p])
		}
	}
}

func TestRecordDesc_AllowsWhitespaceInContent(t *testing.T) {
	d, err := NewDescRound(1, []string{"alice"})
	if err != nil {
		t.Fatalf("NewDescRound error: %v", err)
	}

	err = d.RecordDesc("alice", "  has leading and trailing  ")
	if err != nil {
		t.Fatalf("RecordDesc with whitespace content: err = %v", err)
	}
	// The description is stored as-is (not trimmed).
	if !strings.Contains(d.Descriptions["alice"], "leading") {
		t.Errorf("Descriptions[alice] = %q, should contain original content", d.Descriptions["alice"])
	}
}

func TestCheckWinCondition(t *testing.T) {
	tests := []struct {
		name       string
		players    []*Player
		wantWinner Role
		wantOver   bool
	}{
		{
			name: "undercover eliminated - civilian wins",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: true},
				{Name: "bob", Role: Civilian, Alive: true},
				{Name: "carol", Role: Civilian, Alive: true},
				{Name: "dave", Role: Undercover, Alive: false},
			},
			wantWinner: Civilian,
			wantOver:   true,
		},
		{
			name: "undercover outnumbers civilians - undercover wins",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: true},
				{Name: "bob", Role: Civilian, Alive: false},
				{Name: "carol", Role: Civilian, Alive: false},
				{Name: "dave", Role: Undercover, Alive: true},
			},
			wantWinner: Undercover,
			wantOver:   true,
		},
		{
			name: "undercover equals civilians - undercover wins",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: true},
				{Name: "bob", Role: Civilian, Alive: true},
				{Name: "carol", Role: Undercover, Alive: true},
				{Name: "dave", Role: Undercover, Alive: true},
			},
			wantWinner: Undercover,
			wantOver:   true,
		},
		{
			name: "undercover alive fewer than civilians - continue",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: true},
				{Name: "bob", Role: Civilian, Alive: true},
				{Name: "carol", Role: Civilian, Alive: true},
				{Name: "dave", Role: Civilian, Alive: true},
				{Name: "eve", Role: Undercover, Alive: true},
			},
			wantWinner: 0,
			wantOver:   false,
		},
		{
			name: "all civilians - civilian wins",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: true},
				{Name: "bob", Role: Civilian, Alive: true},
				{Name: "carol", Role: Civilian, Alive: true},
			},
			wantWinner: Civilian,
			wantOver:   true,
		},
		{
			name:       "no players - civilian wins",
			players:    []*Player{},
			wantWinner: Civilian,
			wantOver:   true,
		},
		{
			name: "blank counts as civilian - undercover eliminated",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: true},
				{Name: "bob", Role: Blank, Alive: true},
				{Name: "carol", Role: Undercover, Alive: false},
			},
			wantWinner: Civilian,
			wantOver:   true,
		},
		{
			name: "blank counts as civilian - undercover wins",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: false},
				{Name: "bob", Role: Blank, Alive: true},
				{Name: "carol", Role: Undercover, Alive: true},
				{Name: "dave", Role: Undercover, Alive: true},
			},
			wantWinner: Undercover,
			wantOver:   true,
		},
		{
			name: "all dead except undercover - undercover wins",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: false},
				{Name: "bob", Role: Civilian, Alive: false},
				{Name: "carol", Role: Undercover, Alive: true},
			},
			wantWinner: Undercover,
			wantOver:   true,
		},
		{
			name: "single undercover alive single civilian alive - undercover wins",
			players: []*Player{
				{Name: "alice", Role: Civilian, Alive: true},
				{Name: "bob", Role: Undercover, Alive: true},
			},
			wantWinner: Undercover,
			wantOver:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWinner, gotOver := CheckWinCondition(tt.players)
			if gotWinner != tt.wantWinner {
				t.Errorf("CheckWinCondition() winner = %v, want %v", gotWinner, tt.wantWinner)
			}
			if gotOver != tt.wantOver {
				t.Errorf("CheckWinCondition() gameOver = %v, want %v", gotOver, tt.wantOver)
			}
		})
	}
}
