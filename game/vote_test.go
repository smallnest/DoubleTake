package game

import (
	"strings"
	"testing"
)

// ---------- NewVoteRound ----------

func TestNewVoteRound(t *testing.T) {
	players := []string{"alice", "bob", "carol"}
	v, err := NewVoteRound(1, players)
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	if v.RoundNum != 1 {
		t.Errorf("RoundNum = %d, want 1", v.RoundNum)
	}
	if len(v.Voters) != 3 {
		t.Fatalf("len(Voters) = %d, want 3", len(v.Voters))
	}
	if v.CurrentIndex != 0 {
		t.Errorf("CurrentIndex = %d, want 0", v.CurrentIndex)
	}
	if len(v.Votes) != 0 {
		t.Errorf("Votes should be empty, got %d entries", len(v.Votes))
	}
	// Verify the original slice is not shared.
	players[0] = "changed"
	if v.Voters[0] == "changed" {
		t.Error("Voters shares backing array with input slice")
	}
}

func TestNewVoteRound_Errors(t *testing.T) {
	tests := []struct {
		name        string
		players     []string
		wantErrSub  string
	}{
		{"empty name", []string{"alice", "", "carol"}, "must not be empty"},
		{"whitespace name", []string{"alice", "  ", "carol"}, "must not be empty"},
		{"duplicate name", []string{"alice", "bob", "alice"}, "duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVoteRound(1, tt.players)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

// ---------- CurrentVoter ----------

func TestCurrentVoter(t *testing.T) {
	v, err := NewVoteRound(2, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	if got := v.CurrentVoter(); got != "alice" {
		t.Errorf("CurrentVoter() = %q, want %q", got, "alice")
	}
}

func TestCurrentVoter_AllDone(t *testing.T) {
	v, err := NewVoteRound(1, []string{"alice"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}
	v.CurrentIndex = 1

	if got := v.CurrentVoter(); got != "" {
		t.Errorf("CurrentVoter() = %q, want empty string when all done", got)
	}
}

// ---------- RecordVote normal flow ----------

func TestRecordVote_NormalFlow(t *testing.T) {
	alive := []string{"alice", "bob", "carol"}
	v, err := NewVoteRound(1, alive)
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	if err := v.RecordVote("alice", "bob", alive); err != nil {
		t.Fatalf("RecordVote(alice→bob) error: %v", err)
	}
	if v.CurrentVoter() != "bob" {
		t.Errorf("after alice, CurrentVoter() = %q, want %q", v.CurrentVoter(), "bob")
	}
	if v.Votes["alice"] != "bob" {
		t.Errorf("Votes[alice] = %q, want %q", v.Votes["alice"], "bob")
	}

	if err := v.RecordVote("bob", "carol", alive); err != nil {
		t.Fatalf("RecordVote(bob→carol) error: %v", err)
	}
	if v.CurrentVoter() != "carol" {
		t.Errorf("after bob, CurrentVoter() = %q, want %q", v.CurrentVoter(), "carol")
	}

	if err := v.RecordVote("carol", "alice", alive); err != nil {
		t.Fatalf("RecordVote(carol→alice) error: %v", err)
	}
	if !v.AllDone() {
		t.Error("AllDone() = false, want true after all players voted")
	}
	if v.CurrentVoter() != "" {
		t.Errorf("CurrentVoter() = %q after all done, want empty", v.CurrentVoter())
	}
}

// ---------- RecordVote validation errors ----------

func TestRecordVote_EmptyTarget(t *testing.T) {
	v, err := NewVoteRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	tests := []struct {
		name   string
		target string
	}{
		{"empty string", ""},
		{"spaces only", "   "},
		{"tabs only", "\t\t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.RecordVote("alice", tt.target, []string{"alice", "bob"})
			if err != ErrVoteEmpty {
				t.Errorf("RecordVote(%q) err = %v, want ErrVoteEmpty", tt.name, err)
			}
		})
	}
	// Verify index hasn't advanced.
	if v.CurrentIndex != 0 {
		t.Errorf("CurrentIndex = %d after empty vote rejection, want 0", v.CurrentIndex)
	}
}

func TestRecordVote_NotYourTurn(t *testing.T) {
	v, err := NewVoteRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	err = v.RecordVote("bob", "alice", []string{"alice", "bob"})
	if err != ErrNotYourTurn {
		t.Errorf("RecordVote(bob) err = %v, want ErrNotYourTurn", err)
	}

	// After all done, any player gets ErrNotYourTurn.
	v2, err2 := NewVoteRound(1, []string{"alice"})
	if err2 != nil {
		t.Fatalf("NewVoteRound error: %v", err2)
	}
	_ = v2.RecordVote("alice", "bob", []string{"alice", "bob"})
	err = v2.RecordVote("alice", "carol", []string{"alice", "carol"})
	if err != ErrNotYourTurn {
		t.Errorf("RecordVote after all done: err = %v, want ErrNotYourTurn", err)
	}
}

func TestRecordVote_VoteSelf(t *testing.T) {
	v, err := NewVoteRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	err = v.RecordVote("alice", "alice", []string{"alice", "bob"})
	if err != ErrVoteSelf {
		t.Errorf("RecordVote(alice→alice) err = %v, want ErrVoteSelf", err)
	}
	if v.CurrentIndex != 0 {
		t.Errorf("CurrentIndex = %d after self-vote rejection, want 0", v.CurrentIndex)
	}
}

func TestRecordVote_VoteEliminated(t *testing.T) {
	// 3 players start the round, but carol is eliminated mid-round.
	// alivePlayers no longer contains carol → ErrVoteEliminated.
	v, err := NewVoteRound(1, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	err = v.RecordVote("alice", "carol", []string{"alice", "bob"})
	if err != ErrVoteEliminated {
		t.Errorf("RecordVote(alice→carol) err = %v, want ErrVoteEliminated", err)
	}
	if v.CurrentIndex != 0 {
		t.Errorf("CurrentIndex = %d after eliminated-vote rejection, want 0", v.CurrentIndex)
	}
}

func TestRecordVote_VoteUnknown(t *testing.T) {
	v, err := NewVoteRound(1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	// "nobody" is not a voter and not in alive list → unknown
	err = v.RecordVote("alice", "nobody", []string{"alice", "bob"})
	if err != ErrVoteUnknown {
		t.Errorf("RecordVote(alice→nobody) err = %v, want ErrVoteUnknown", err)
	}
}

// ---------- AllDone boundary cases ----------

func TestAllDone_ZeroPlayers(t *testing.T) {
	v, err := NewVoteRound(1, []string{})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}
	if !v.AllDone() {
		t.Error("AllDone() = false for 0 players, want true")
	}
	if got := v.CurrentVoter(); got != "" {
		t.Errorf("CurrentVoter() = %q for 0 players, want empty", got)
	}
}

func TestAllDone_OnePlayer_VoteRound(t *testing.T) {
	v, err := NewVoteRound(1, []string{"solo"})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}
	if v.AllDone() {
		t.Error("AllDone() = true before solo player votes")
	}
	// solo must vote for someone else (they can't, but let's test with an
	// external target in the alive list)
	if err := v.RecordVote("solo", "other", []string{"solo", "other"}); err != nil {
		t.Fatalf("RecordVote(solo) error: %v", err)
	}
	if !v.AllDone() {
		t.Error("AllDone() = false after solo player votes")
	}
}

// ---------- Tally ----------

func TestTally(t *testing.T) {
	alive := []string{"alice", "bob", "carol", "dave"}
	v, err := NewVoteRound(1, alive)
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	_ = v.RecordVote("alice", "bob", alive)
	_ = v.RecordVote("bob", "carol", alive)
	_ = v.RecordVote("carol", "bob", alive)
	_ = v.RecordVote("dave", "bob", alive)

	tally := v.Tally()
	if tally["bob"] != 3 {
		t.Errorf("tally[bob] = %d, want 3", tally["bob"])
	}
	if tally["carol"] != 1 {
		t.Errorf("tally[carol] = %d, want 1", tally["carol"])
	}
	if _, ok := tally["alice"]; ok {
		t.Error("tally should not contain alice (0 votes)")
	}
}

func TestTally_Empty(t *testing.T) {
	v, err := NewVoteRound(1, []string{})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}
	tally := v.Tally()
	if len(tally) != 0 {
		t.Errorf("tally should be empty, got %d entries", len(tally))
	}
}

// ---------- FindEliminated ----------

func TestFindEliminated_ClearWinner(t *testing.T) {
	alive := []string{"alice", "bob", "carol"}
	v, err := NewVoteRound(1, alive)
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	_ = v.RecordVote("alice", "bob", alive)
	_ = v.RecordVote("bob", "carol", alive)
	_ = v.RecordVote("carol", "bob", alive)

	name, tie := v.FindEliminated()
	if tie {
		t.Error("FindEliminated() tie = true, want false")
	}
	if name != "bob" {
		t.Errorf("FindEliminated() = %q, want %q", name, "bob")
	}
}

func TestFindEliminated_Tie(t *testing.T) {
	alive := []string{"alice", "bob", "carol", "dave"}
	v, err := NewVoteRound(1, alive)
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	_ = v.RecordVote("alice", "bob", alive)
	_ = v.RecordVote("bob", "carol", alive)
	_ = v.RecordVote("carol", "bob", alive)
	_ = v.RecordVote("dave", "carol", alive)

	name, tie := v.FindEliminated()
	if !tie {
		t.Error("FindEliminated() tie = false, want true for tied votes")
	}
	if name != "" {
		t.Errorf("FindEliminated() = %q on tie, want empty string", name)
	}
}

func TestFindEliminated_AllTie(t *testing.T) {
	alive := []string{"alice", "bob", "carol"}
	v, err := NewVoteRound(1, alive)
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	// Each person votes for the next → each gets 1 vote → tie
	_ = v.RecordVote("alice", "bob", alive)
	_ = v.RecordVote("bob", "carol", alive)
	_ = v.RecordVote("carol", "alice", alive)

	name, tie := v.FindEliminated()
	if !tie {
		t.Error("FindEliminated() tie = false, want true for all-tied votes")
	}
	if name != "" {
		t.Errorf("FindEliminated() = %q on all-tie, want empty string", name)
	}
}

func TestFindEliminated_NoVotes(t *testing.T) {
	v, err := NewVoteRound(1, []string{})
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	name, tie := v.FindEliminated()
	if name != "" {
		t.Errorf("FindEliminated() = %q for no votes, want empty", name)
	}
	if tie {
		t.Error("FindEliminated() tie = true for no votes, want false")
	}
}

// ---------- Full round integration ----------

func TestVoteRound_FullRound_Integration(t *testing.T) {
	// Simulate a 5-player round: alice, bob, carol, dave, eve
	// Votes: alice→bob, bob→carol, carol→bob, dave→bob, eve→carol
	// Result: bob has 3 votes, carol has 2, rest 0
	alive := []string{"alice", "bob", "carol", "dave", "eve"}
	v, err := NewVoteRound(3, alive)
	if err != nil {
		t.Fatalf("NewVoteRound error: %v", err)
	}

	votes := map[string]string{
		"alice": "bob",
		"bob":   "carol",
		"carol": "bob",
		"dave":  "bob",
		"eve":   "carol",
	}
	for _, voter := range alive {
		if err := v.RecordVote(voter, votes[voter], alive); err != nil {
			t.Fatalf("RecordVote(%s→%s) error: %v", voter, votes[voter], err)
		}
	}

	if !v.AllDone() {
		t.Error("AllDone() = false after all votes recorded")
	}

	tally := v.Tally()
	if tally["bob"] != 3 {
		t.Errorf("tally[bob] = %d, want 3", tally["bob"])
	}
	if tally["carol"] != 2 {
		t.Errorf("tally[carol] = %d, want 2", tally["carol"])
	}

	eliminated, tie := v.FindEliminated()
	if tie {
		t.Error("FindEliminated() tie = true, want false")
	}
	if eliminated != "bob" {
		t.Errorf("FindEliminated() = %q, want %q", eliminated, "bob")
	}
}
