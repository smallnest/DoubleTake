package game

import (
	"strings"
	"testing"
)

// ---------- NewPKRound ----------

func TestNewPKRound(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	if pk.Num != 1 {
		t.Errorf("Num = %d, want 1", pk.Num)
	}
	if len(pk.Tied) != 2 {
		t.Errorf("len(Tied) = %d, want 2", len(pk.Tied))
	}
	if pk.Phase != "desc" {
		t.Errorf("Phase = %q, want %q", pk.Phase, "desc")
	}
	if len(pk.VoterOrder) != 3 {
		t.Errorf("len(VoterOrder) = %d, want 3", len(pk.VoterOrder))
	}
	if len(pk.Votes) != 0 {
		t.Errorf("Votes should be empty, got %d entries", len(pk.Votes))
	}
}

func TestNewPKRound_Errors(t *testing.T) {
	tests := []struct {
		name       string
		tied       []string
		alive      []string
		wantErrSub string
	}{
		{"too few tied", []string{"alice"}, []string{"alice", "bob"}, "at least 2"},
		{"empty tied name", []string{"alice", ""}, []string{"alice", "bob"}, "must not be empty"},
		{"duplicate tied", []string{"alice", "alice"}, []string{"alice", "bob"}, "duplicate"},
		{"tied not alive", []string{"alice", "bob"}, []string{"alice", "carol"}, "not alive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPKRound(1, tt.tied, tt.alive)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

// ---------- PK Description Phase ----------

func TestPKRound_DescPhase(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}

	// First speaker should be alice (first in tied list).
	if got := pk.CurrentSpeaker(); got != "alice" {
		t.Errorf("CurrentSpeaker() = %q, want %q", got, "alice")
	}

	if err := pk.RecordDesc("alice", "my description"); err != nil {
		t.Fatalf("RecordDesc(alice) error: %v", err)
	}
	if got := pk.CurrentSpeaker(); got != "bob" {
		t.Errorf("CurrentSpeaker() = %q, want %q", got, "bob")
	}

	if err := pk.RecordDesc("bob", "bob desc"); err != nil {
		t.Fatalf("RecordDesc(bob) error: %v", err)
	}
	if !pk.Desc.AllDone() {
		t.Error("AllDone() = false, want true after all tied players described")
	}
}

func TestPKRound_NonTiedPlayerCannotDescribe(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}

	err = pk.RecordDesc("carol", "I want to speak")
	if err != ErrNotYourTurn {
		t.Errorf("RecordDesc(carol) err = %v, want ErrNotYourTurn", err)
	}
}

// ---------- PK Vote Phase ----------

func TestPKRound_VotePhase(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}

	pk.StartVote()
	if pk.Phase != "vote" {
		t.Errorf("Phase = %q, want %q", pk.Phase, "vote")
	}

	// First voter is alice (first in alive list).
	if got := pk.CurrentVoter(); got != "alice" {
		t.Errorf("CurrentVoter() = %q, want %q", got, "alice")
	}

	// alice votes for bob (a tied player).
	if err := pk.RecordVote("alice", "bob", alive); err != nil {
		t.Fatalf("RecordVote(alice→bob) error: %v", err)
	}

	// bob votes for alice.
	if err := pk.RecordVote("bob", "alice", alive); err != nil {
		t.Fatalf("RecordVote(bob→alice) error: %v", err)
	}

	// carol votes for bob.
	if err := pk.RecordVote("carol", "bob", alive); err != nil {
		t.Fatalf("RecordVote(carol→bob) error: %v", err)
	}

	if !pk.AllVoted() {
		t.Error("AllVoted() = false, want true after all alive players voted")
	}
}

func TestPKRound_CannotVoteForNonTied(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	// carol tries to vote for herself (not tied) — should fail.
	err = pk.RecordVote("alice", "carol", alive)
	if err != ErrVoteNotTied {
		t.Errorf("RecordVote(alice→carol) err = %v, want ErrVoteNotTied", err)
	}
}

func TestPKRound_VoteSelf(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	err = pk.RecordVote("alice", "alice", alive)
	if err != ErrVoteSelf {
		t.Errorf("RecordVote(alice→alice) err = %v, want ErrVoteSelf", err)
	}
}

func TestPKRound_VoteEmpty(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	err = pk.RecordVote("alice", "  ", alive)
	if err != ErrVoteEmpty {
		t.Errorf("RecordVote(alice→empty) err = %v, want ErrVoteEmpty", err)
	}
}

func TestPKRound_VoteNotYourTurn(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	// bob tries to vote before alice.
	err = pk.RecordVote("bob", "alice", alive)
	if err != ErrNotYourTurn {
		t.Errorf("RecordVote(bob) err = %v, want ErrNotYourTurn", err)
	}
}

func TestPKRound_VoteEliminated(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	// dave is not alive — should get ErrVoteEliminated.
	err = pk.RecordVote("alice", "dave", []string{"alice", "bob"})
	if err != ErrVoteEliminated {
		t.Errorf("RecordVote(alice→dave) err = %v, want ErrVoteEliminated", err)
	}
}

// ---------- Tally and FindEliminated ----------

func TestPKRound_TallyAndFindEliminated(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	_ = pk.RecordVote("alice", "bob", alive)
	_ = pk.RecordVote("bob", "alice", alive)
	_ = pk.RecordVote("carol", "bob", alive)

	tally := pk.Tally()
	if tally["bob"] != 2 {
		t.Errorf("tally[bob] = %d, want 2", tally["bob"])
	}
	if tally["alice"] != 1 {
		t.Errorf("tally[alice] = %d, want 1", tally["alice"])
	}

	eliminated, tie := pk.FindEliminated()
	if tie {
		t.Error("FindEliminated() tie = true, want false")
	}
	if eliminated != "bob" {
		t.Errorf("FindEliminated() = %q, want %q", eliminated, "bob")
	}
}

func TestPKRound_FindEliminated_StillTied(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	// alice→bob, bob→alice, carol→alice → alice:2, bob:1 → alice eliminated
	// But let's make it tied: alice→bob, bob→alice, carol abstains? No, all must vote.
	// alice→bob, bob→alice, carol→bob → bob:2, alice:1 → bob eliminated.
	// To get a tie: alice→bob, bob→alice, carol→alice → alice:2, bob:1.
	// Actually 3 voters, 2 tied: can't tie with odd voters.
	// Let's use 4 alive players.
	tied2 := []string{"alice", "bob"}
	alive2 := []string{"alice", "bob", "carol", "dave"}
	pk2, err2 := NewPKRound(1, tied2, alive2)
	if err2 != nil {
		t.Fatalf("NewPKRound error: %v", err2)
	}
	pk2.StartVote()

	_ = pk2.RecordVote("alice", "bob", alive2)
	_ = pk2.RecordVote("bob", "alice", alive2)
	_ = pk2.RecordVote("carol", "alice", alive2)
	_ = pk2.RecordVote("dave", "bob", alive2)

	// alice:2, bob:2 → tie
	eliminated, tie := pk2.FindEliminated()
	if !tie {
		t.Error("FindEliminated() tie = false, want true")
	}
	if eliminated != "" {
		t.Errorf("FindEliminated() = %q, want empty on tie", eliminated)
	}
}

func TestPKRound_FindEliminated_NoVotes(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	eliminated, tie := pk.FindEliminated()
	if eliminated != "" {
		t.Errorf("FindEliminated() = %q, want empty", eliminated)
	}
	if tie {
		t.Error("FindEliminated() tie = true for no votes, want false")
	}
}

// ---------- Full PK integration ----------

func TestPKRound_FullRound_Integration(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol", "dave"}
	pk, err := NewPKRound(2, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}

	// Description phase.
	if err := pk.RecordDesc("alice", "I am alice"); err != nil {
		t.Fatalf("RecordDesc(alice) error: %v", err)
	}
	if err := pk.RecordDesc("bob", "I am bob"); err != nil {
		t.Fatalf("RecordDesc(bob) error: %v", err)
	}
	if !pk.Desc.AllDone() {
		t.Fatal("desc phase not done")
	}

	// Vote phase.
	pk.StartVote()
	if err := pk.RecordVote("alice", "bob", alive); err != nil {
		t.Fatalf("RecordVote error: %v", err)
	}
	if err := pk.RecordVote("bob", "alice", alive); err != nil {
		t.Fatalf("RecordVote error: %v", err)
	}
	if err := pk.RecordVote("carol", "bob", alive); err != nil {
		t.Fatalf("RecordVote error: %v", err)
	}
	if err := pk.RecordVote("dave", "bob", alive); err != nil {
		t.Fatalf("RecordVote error: %v", err)
	}

	if !pk.AllVoted() {
		t.Fatal("AllVoted() = false")
	}

	tally := pk.Tally()
	if tally["bob"] != 3 {
		t.Errorf("tally[bob] = %d, want 3", tally["bob"])
	}
	if tally["alice"] != 1 {
		t.Errorf("tally[alice] = %d, want 1", tally["alice"])
	}

	eliminated, tie := pk.FindEliminated()
	if tie {
		t.Fatal("FindEliminated() tie = true, want false")
	}
	if eliminated != "bob" {
		t.Errorf("FindEliminated() = %q, want %q", eliminated, "bob")
	}
}

// ---------- SkipCurrentVoter ----------

func TestSkipCurrentVoter(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	if pk.CurrentVoter() != "alice" {
		t.Fatalf("expected alice as first voter, got %q", pk.CurrentVoter())
	}

	pk.SkipCurrentVoter()
	if pk.CurrentVoter() != "bob" {
		t.Errorf("after skip, expected bob, got %q", pk.CurrentVoter())
	}
	if len(pk.Votes) != 0 {
		t.Error("SkipCurrentVoter should not record a vote")
	}
}

func TestSkipCurrentVoter_AllSkipped(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()

	pk.SkipCurrentVoter()
	pk.SkipCurrentVoter()
	pk.SkipCurrentVoter()
	if !pk.AllVoted() {
		t.Error("AllVoted() should be true after skipping all voters")
	}
}

func TestSkipCurrentVoter_NoOpWhenDone(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.StartVote()
	pk.CurrentVote = 3 // all done

	pk.SkipCurrentVoter() // should be a no-op
	if pk.CurrentVote != 3 {
		t.Errorf("CurrentVote should remain 3 after no-op skip, got %d", pk.CurrentVote)
	}
}
