package game

import (
	"strings"
	"testing"
)

func TestNewDescRound(t *testing.T) {
	players := []string{"alice", "bob", "carol"}
	d := NewDescRound(1, players)

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
	d := NewDescRound(2, []string{"alice", "bob", "carol"})

	if got := d.CurrentSpeaker(); got != "alice" {
		t.Errorf("CurrentSpeaker() = %q, want %q", got, "alice")
	}
}

func TestCurrentSpeaker_AllDone(t *testing.T) {
	d := NewDescRound(1, []string{"alice"})
	d.CurrentIndex = 1

	if got := d.CurrentSpeaker(); got != "" {
		t.Errorf("CurrentSpeaker() = %q, want empty string when all done", got)
	}
}

func TestRecordDesc_NormalFlow(t *testing.T) {
	d := NewDescRound(1, []string{"alice", "bob", "carol"})

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
	d := NewDescRound(1, []string{"alice", "bob"})

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
	d := NewDescRound(1, []string{"alice", "bob"})

	err := d.RecordDesc("bob", "something")
	if err != ErrNotYourTurn {
		t.Errorf("RecordDesc(bob) err = %v, want ErrNotYourTurn", err)
	}

	// After all done, any player gets ErrNotYourTurn.
	d2 := NewDescRound(1, []string{"alice"})
	_ = d2.RecordDesc("alice", "ok")
	err = d2.RecordDesc("alice", "another")
	if err != ErrNotYourTurn {
		t.Errorf("RecordDesc after all done: err = %v, want ErrNotYourTurn", err)
	}
}

func TestAllDone_EmptyPlayers(t *testing.T) {
	d := NewDescRound(1, []string{})
	if !d.AllDone() {
		t.Error("AllDone() = false for 0 players, want true")
	}
	if got := d.CurrentSpeaker(); got != "" {
		t.Errorf("CurrentSpeaker() = %q for 0 players, want empty", got)
	}
}

func TestAllDone_OnePlayer(t *testing.T) {
	d := NewDescRound(1, []string{"solo"})
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
	d := NewDescRound(3, players)

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
	d := NewDescRound(1, []string{"alice"})

	err := d.RecordDesc("alice", "  has leading and trailing  ")
	if err != nil {
		t.Fatalf("RecordDesc with whitespace content: err = %v", err)
	}
	// The description is stored as-is (not trimmed).
	if !strings.Contains(d.Descriptions["alice"], "leading") {
		t.Errorf("Descriptions[alice] = %q, should contain original content", d.Descriptions["alice"])
	}
}
