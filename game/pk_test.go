package game

import (
	"strings"
	"testing"
)

// ---------- NewPKRound validation ----------

func TestNewPKRound_Success(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	if pk.PKNum != 1 {
		t.Errorf("PKNum = %d, want 1", pk.PKNum)
	}
	if len(pk.TiedPlayers) != 2 {
		t.Errorf("len(TiedPlayers) = %d, want 2", len(pk.TiedPlayers))
	}
	if len(pk.VoterOrder) != 3 {
		t.Errorf("len(VoterOrder) = %d, want 3", len(pk.VoterOrder))
	}
	if pk.Desc == nil {
		t.Error("Desc should not be nil")
	}
	if len(pk.Votes) != 0 {
		t.Errorf("Votes should be empty, got %d entries", len(pk.Votes))
	}
	// Verify slice copy.
	tied[0] = "changed"
	if pk.TiedPlayers[0] == "changed" {
		t.Error("TiedPlayers shares backing array with input slice")
	}
}

func TestNewPKRound_EmptyTiedPlayers(t *testing.T) {
	_, err := NewPKRound(1, []string{}, []string{"alice"})
	if err == nil || !strings.Contains(err.Error(), "tied players list must not be empty") {
		t.Errorf("expected tied players empty error, got: %v", err)
	}
}

func TestNewPKRound_DuplicateTiedPlayers(t *testing.T) {
	_, err := NewPKRound(1, []string{"alice", "alice"}, []string{"alice", "bob"})
	if err == nil || !strings.Contains(err.Error(), "duplicate tied player") {
		t.Errorf("expected duplicate tied player error, got: %v", err)
	}
}

func TestNewPKRound_EmptyTiedPlayerName(t *testing.T) {
	_, err := NewPKRound(1, []string{"", "bob"}, []string{"bob"})
	if err == nil || !strings.Contains(err.Error(), "tied player name must not be empty") {
		t.Errorf("expected empty tied player name error, got: %v", err)
	}
}

func TestNewPKRound_EmptyAlivePlayerName(t *testing.T) {
	_, err := NewPKRound(1, []string{"alice"}, []string{"alice", ""})
	if err == nil || !strings.Contains(err.Error(), "alive player name must not be empty") {
		t.Errorf("expected empty alive player name error, got: %v", err)
	}
}

func TestNewPKRound_DuplicateAlivePlayers(t *testing.T) {
	_, err := NewPKRound(1, []string{"alice"}, []string{"alice", "alice"})
	if err == nil || !strings.Contains(err.Error(), "duplicate alive player") {
		t.Errorf("expected duplicate alive player error, got: %v", err)
	}
}

// ---------- Description phase ----------

func TestPKRound_DescriptionPhase(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}

	// Current speaker should be first tied player.
	if got := pk.CurrentSpeaker(); got != "alice" {
		t.Errorf("CurrentSpeaker() = %q, want %q", got, "alice")
	}

	// Record description for alice.
	if err := pk.RecordPKDesc("alice", "it is sweet"); err != nil {
		t.Fatalf("RecordPKDesc(alice) error: %v", err)
	}
	if got := pk.CurrentSpeaker(); got != "bob" {
		t.Errorf("after alice, CurrentSpeaker() = %q, want %q", got, "bob")
	}

	// Record description for bob.
	if err := pk.RecordPKDesc("bob", "it is sour"); err != nil {
		t.Fatalf("RecordPKDesc(bob) error: %v", err)
	}
	if !pk.DescAllDone() {
		t.Error("DescAllDone() = false, want true")
	}
	if pk.CurrentSpeaker() != "" {
		t.Errorf("CurrentSpeaker() = %q after all done, want empty", pk.CurrentSpeaker())
	}

	// Verify descriptions.
	desc, ok := pk.Description("alice")
	if !ok || desc != "it is sweet" {
		t.Errorf("Description(alice) = %q, ok=%v, want %q, true", desc, ok, "it is sweet")
	}
	desc, ok = pk.Description("bob")
	if !ok || desc != "it is sour" {
		t.Errorf("Description(bob) = %q, ok=%v, want %q, true", desc, ok, "it is sour")
	}
}

func TestPKRound_DescRejectsEmptyDesc(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice"}, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	if err := pk.RecordPKDesc("alice", "  "); err != ErrEmptyDesc {
		t.Errorf("RecordPKDesc with empty desc: err = %v, want ErrEmptyDesc", err)
	}
}

func TestPKRound_DescRejectsNotYourTurn(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	if err := pk.RecordPKDesc("bob", "something"); err != ErrNotYourTurn {
		t.Errorf("RecordPKDesc(bob) when alice's turn: err = %v, want ErrNotYourTurn", err)
	}
}

// ---------- Voting phase ----------

func TestPKRound_NormalVoteFlow(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol"}
	pk, err := NewPKRound(1, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}

	// First voter.
	if got := pk.CurrentVoter(); got != "alice" {
		t.Errorf("CurrentVoter() = %q, want %q", got, "alice")
	}

	if err := pk.RecordPKVote("alice", "bob"); err != nil {
		t.Fatalf("RecordPKVote(alice→bob) error: %v", err)
	}

	if got := pk.CurrentVoter(); got != "bob" {
		t.Errorf("after alice votes, CurrentVoter() = %q, want %q", got, "bob")
	}

	if err := pk.RecordPKVote("bob", "alice"); err != nil {
		t.Fatalf("RecordPKVote(bob→alice) error: %v", err)
	}
	if err := pk.RecordPKVote("carol", "bob"); err != nil {
		t.Fatalf("RecordPKVote(carol→bob) error: %v", err)
	}

	if !pk.AllVotesDone() {
		t.Error("AllVotesDone() = false, want true")
	}
	if pk.CurrentVoter() != "" {
		t.Errorf("CurrentVoter() = %q after all votes, want empty", pk.CurrentVoter())
	}
}

func TestPKRound_VoteRejectsNonTiedTarget(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	err = pk.RecordPKVote("carol", "dave")
	if err != ErrPKVoteTargetInvalid {
		t.Errorf("vote for non-tied player: err = %v, want ErrPKVoteTargetInvalid", err)
	}
}

func TestPKRound_VoteRejectsEmptyTarget(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	err = pk.RecordPKVote("carol", "")
	if err != ErrPKVoteEmpty {
		t.Errorf("vote with empty target: err = %v, want ErrPKVoteEmpty", err)
	}
	err = pk.RecordPKVote("carol", "   ")
	if err != ErrPKVoteEmpty {
		t.Errorf("vote with whitespace target: err = %v, want ErrPKVoteEmpty", err)
	}
}

func TestPKRound_VoteRejectsNonAliveVoter(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	err = pk.RecordPKVote("dave", "alice")
	if err != ErrPKNotAliveVoter {
		t.Errorf("vote from non-alive voter: err = %v, want ErrPKNotAliveVoter", err)
	}
}

func TestPKRound_VoteRejectsDoubleVote(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	if err := pk.RecordPKVote("alice", "bob"); err != nil {
		t.Fatalf("first vote error: %v", err)
	}
	err = pk.RecordPKVote("alice", "alice")
	if err != ErrPKAlreadyVoted {
		t.Errorf("double vote: err = %v, want ErrPKAlreadyVoted", err)
	}
}

// ---------- Tally and FindEliminated ----------

func TestPKRound_Tally(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.Votes["alice"] = "bob"
	pk.Votes["bob"] = "bob"
	pk.Votes["carol"] = "alice"

	tally := pk.Tally()
	if tally["alice"] != 1 {
		t.Errorf("tally[alice] = %d, want 1", tally["alice"])
	}
	if tally["bob"] != 2 {
		t.Errorf("tally[bob] = %d, want 2", tally["bob"])
	}
}

func TestPKRound_FindEliminated_UniqueWinner(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.Votes["alice"] = "bob"
	pk.Votes["bob"] = "bob"
	pk.Votes["carol"] = "bob"

	player, tie := pk.FindEliminated()
	if tie {
		t.Error("FindEliminated() tie = true, want false")
	}
	if player != "bob" {
		t.Errorf("FindEliminated() player = %q, want %q", player, "bob")
	}
}

func TestPKRound_FindEliminated_Tie(t *testing.T) {
	// Two tied players, each voted for the other: 1-1 tie.
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.Votes["alice"] = "bob"
	pk.Votes["bob"] = "alice"

	player, tie := pk.FindEliminated()
	if !tie {
		t.Error("FindEliminated() tie = false, want true for tied votes")
	}
	if player != "" {
		t.Errorf("FindEliminated() player = %q, want empty string on tie", player)
	}
}

func TestPKRound_FindEliminated_ThreeWayTie(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob", "carol"}, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	pk.Votes["alice"] = "alice"
	pk.Votes["bob"] = "bob"
	pk.Votes["carol"] = "carol"

	player, tie := pk.FindEliminated()
	if !tie {
		t.Error("FindEliminated() tie = false, want true for 3-way tie")
	}
	if player != "" {
		t.Errorf("FindEliminated() player = %q, want empty on tie", player)
	}
}

func TestPKRound_FindEliminated_NoVotes(t *testing.T) {
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	// No votes recorded — tie.
	player, tie := pk.FindEliminated()
	if !tie {
		t.Error("FindEliminated() with no votes: tie = false, want true")
	}
	if player != "" {
		t.Errorf("FindEliminated() with no votes: player = %q, want empty", player)
	}
}

// ---------- New validation tests ----------

func TestNewPKRound_NilAlivePlayers(t *testing.T) {
	_, err := NewPKRound(1, []string{"alice"}, nil)
	if err == nil || !strings.Contains(err.Error(), "alive players list must not be empty") {
		t.Errorf("expected alive players empty error, got: %v", err)
	}
}

func TestNewPKRound_EmptyAlivePlayers(t *testing.T) {
	_, err := NewPKRound(1, []string{"alice"}, []string{})
	if err == nil || !strings.Contains(err.Error(), "alive players list must not be empty") {
		t.Errorf("expected alive players empty error, got: %v", err)
	}
}

func TestNewPKRound_TiedPlayerNotInAlive(t *testing.T) {
	// alice is tied but not in alive players list.
	_, err := NewPKRound(1, []string{"alice"}, []string{"bob"})
	if err == nil || !strings.Contains(err.Error(), "tied player") {
		t.Errorf("expected tied-not-in-alive error, got: %v", err)
	}
}

func TestPKRound_SelfVoteAllowed(t *testing.T) {
	// A tied player can vote for themselves.
	pk, err := NewPKRound(1, []string{"alice", "bob"}, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}
	if err := pk.RecordPKVote("alice", "alice"); err != nil {
		t.Fatalf("RecordPKVote(alice→alice) error: %v", err)
	}
	if err := pk.RecordPKVote("bob", "alice"); err != nil {
		t.Fatalf("RecordPKVote(bob→alice) error: %v", err)
	}
	player, tie := pk.FindEliminated()
	if tie {
		t.Error("FindEliminated() tie = true, want false")
	}
	if player != "alice" {
		t.Errorf("FindEliminated() player = %q, want %q", player, "alice")
	}
}

// ---------- Full PK flow (description + voting) ----------



func TestPKRound_FullFlow(t *testing.T) {
	tied := []string{"alice", "bob"}
	alive := []string{"alice", "bob", "carol", "dave"}
	pk, err := NewPKRound(2, tied, alive)
	if err != nil {
		t.Fatalf("NewPKRound error: %v", err)
	}

	// Description phase.
	if err := pk.RecordPKDesc("alice", "PK desc from alice"); err != nil {
		t.Fatalf("RecordPKDesc(alice) error: %v", err)
	}
	if err := pk.RecordPKDesc("bob", "PK desc from bob"); err != nil {
		t.Fatalf("RecordPKDesc(bob) error: %v", err)
	}
	if !pk.DescAllDone() {
		t.Fatal("DescAllDone() = false after both described")
	}

	// Voting phase.
	if err := pk.RecordPKVote("alice", "bob"); err != nil {
		t.Fatalf("RecordPKVote(alice→bob) error: %v", err)
	}
	if err := pk.RecordPKVote("bob", "alice"); err != nil {
		t.Fatalf("RecordPKVote(bob→alice) error: %v", err)
	}
	if err := pk.RecordPKVote("carol", "bob"); err != nil {
		t.Fatalf("RecordPKVote(carol→bob) error: %v", err)
	}
	if err := pk.RecordPKVote("dave", "bob"); err != nil {
		t.Fatalf("RecordPKVote(dave→bob) error: %v", err)
	}
	if !pk.AllVotesDone() {
		t.Fatal("AllVotesDone() = false after all voted")
	}

	player, tie := pk.FindEliminated()
	if tie {
		t.Error("FindEliminated() tie = true, want false (bob has 3, alice has 1)")
	}
	if player != "bob" {
		t.Errorf("FindEliminated() player = %q, want %q", player, "bob")
	}
}
