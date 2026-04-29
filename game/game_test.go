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

func TestNewVoteRound(t *testing.T) {
	players := []string{"alice", "bob", "carol"}
	vr, err := NewVoteRound(1, players)
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}
	if vr.RoundNum != 1 {
		t.Errorf("RoundNum = %d, want 1", vr.RoundNum)
	}
	if len(vr.Players) != 3 {
		t.Fatalf("len(Players) = %d, want 3", len(vr.Players))
	}
	if len(vr.Votes) != 0 {
		t.Errorf("Votes should be empty, got %d entries", len(vr.Votes))
	}
	players[0] = "changed"
	if vr.Players[0] == "changed" {
		t.Error("Players shares backing array with input slice")
	}
}

func TestNewVoteRound_EmptyPlayers(t *testing.T) {
	_, err := NewVoteRound(1, []string{})
	if err == nil {
		t.Error("expected error for empty players")
	}
}

func TestNewVoteRound_DuplicateNames(t *testing.T) {
	_, err := NewVoteRound(1, []string{"alice", "bob", "alice"})
	if err == nil {
		t.Error("expected error for duplicate names")
	}
}

func TestNewVoteRound_EmptyName(t *testing.T) {
	_, err := NewVoteRound(1, []string{"alice", ""})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestVoteRound_CurrentVoter(t *testing.T) {
	vr, err := NewVoteRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}
	if got := vr.CurrentVoter(); got != "alice" {
		t.Errorf("CurrentVoter() = %q, want %q", got, "alice")
	}
	vr.Votes["alice"] = "bob"
	if got := vr.CurrentVoter(); got != "bob" {
		t.Errorf("CurrentVoter() = %q after alice voted, want %q", got, "bob")
	}
	vr.Votes["bob"] = "alice"
	if got := vr.CurrentVoter(); got != "" {
		t.Errorf("CurrentVoter() = %q after all voted, want empty", got)
	}
}

func TestVoteRound_RecordVote_Normal(t *testing.T) {
	vr, err := NewVoteRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}
	if err := vr.RecordVote("alice", "bob"); err != nil {
		t.Fatalf("RecordVote error: %v", err)
	}
	if vr.Votes["alice"] != "bob" {
		t.Errorf("Votes[alice] = %q, want %q", vr.Votes["alice"], "bob")
	}
	if vr.AllVoted() {
		t.Error("AllVoted() = true after only 1/2 voted, want false")
	}
	if err := vr.RecordVote("bob", "alice"); err != nil {
		t.Fatalf("RecordVote(bob) error: %v", err)
	}
	if !vr.AllVoted() {
		t.Error("AllVoted() = false after 2/2 voted, want true")
	}
}

func TestVoteRound_RecordVote_EmptyTarget(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob"})
	err := vr.RecordVote("alice", "")
	if err != ErrVoteEmpty {
		t.Errorf("expected ErrVoteEmpty, got %v", err)
	}
}

func TestVoteRound_RecordVote_NotAPlayer(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob"})
	err := vr.RecordVote("eve", "alice")
	if err != ErrVoteNotAPlayer {
		t.Errorf("expected ErrVoteNotAPlayer, got %v", err)
	}
}

func TestVoteRound_RecordVote_TargetNotAPlayer(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob"})
	err := vr.RecordVote("alice", "eve")
	if err != ErrVoteTargetNotAPlayer {
		t.Errorf("expected ErrVoteTargetNotAPlayer, got %v", err)
	}
}

func TestVoteRound_RecordVote_AlreadyVoted(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob"})
	_ = vr.RecordVote("alice", "bob")
	err := vr.RecordVote("alice", "bob")
	if err != ErrVoteAlreadyVoted {
		t.Errorf("expected ErrVoteAlreadyVoted, got %v", err)
	}
}

func TestVoteRound_Tally(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob", "carol"})
	vr.RecordVote("alice", "bob")
	vr.RecordVote("bob", "bob")
	vr.RecordVote("carol", "alice")
	tally := vr.Tally()
	if tally["alice"] != 1 {
		t.Errorf("tally[alice] = %d, want 1", tally["alice"])
	}
	if tally["bob"] != 2 {
		t.Errorf("tally[bob] = %d, want 2", tally["bob"])
	}
	if tally["carol"] != 0 {
		t.Errorf("tally[carol] = %d, want 0", tally["carol"])
	}
}

func TestVoteRound_FindEliminated_Unique(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob"})
	vr.RecordVote("alice", "bob")
	vr.RecordVote("bob", "bob")
	eliminated, tie := vr.FindEliminated()
	if tie {
		t.Error("expected no tie")
	}
	if eliminated != "bob" {
		t.Errorf("eliminated = %q, want %q", eliminated, "bob")
	}
}

func TestVoteRound_FindEliminated_Tie(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob"})
	vr.RecordVote("alice", "bob")
	vr.RecordVote("bob", "alice")
	_, tie := vr.FindEliminated()
	if !tie {
		t.Error("expected tie")
	}
}

func TestVoteRound_FindEliminated_NoVotes(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob"})
	_, tie := vr.FindEliminated()
	if !tie {
		t.Error("expected tie when no votes cast")
	}
}

func TestVoteRound_AllVoted(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob", "carol"})
	if vr.AllVoted() {
		t.Error("AllVoted() = true before any votes")
	}
	vr.RecordVote("alice", "bob")
	if vr.AllVoted() {
		t.Error("AllVoted() = true after 1/3 votes")
	}
	vr.RecordVote("bob", "carol")
	vr.RecordVote("carol", "alice")
	if !vr.AllVoted() {
		t.Error("AllVoted() = false after all voted")
	}
}

func TestVoteRound_FindEliminated_ThreeWayTie(t *testing.T) {
	vr, _ := NewVoteRound(1, []string{"alice", "bob", "carol"})
	vr.RecordVote("alice", "bob")
	vr.RecordVote("bob", "carol")
	vr.RecordVote("carol", "alice")
	_, tie := vr.FindEliminated()
	if !tie {
		t.Error("expected three-way tie")
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
