package game

import (
	"fmt"
	"strings"
	"testing"
)

func TestRole_String(t *testing.T) {
	tests := []struct {
		role Role
		want string
	}{
		{Civilian, "Civilian"},
		{Undercover, "Undercover"},
		{Blank, "Blank"},
		{Role(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.role.String(); got != tt.want {
				t.Errorf("Role.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAssignRoles_Success(t *testing.T) {
	names := []string{"alice", "bob", "carol", "dave", "eve"}
	players, err := AssignRoles(names, 1, 1)
	if err != nil {
		t.Fatalf("AssignRoles() unexpected error: %v", err)
	}
	if len(players) != len(names) {
		t.Fatalf("AssignRoles() returned %d players, want %d", len(players), len(names))
	}

	var undercovers, blanks, civilians int
	for _, p := range players {
		if !p.Alive {
			t.Errorf("player %q: Alive = %v, want true", p.Name, p.Alive)
		}
		if p.Connected {
			t.Errorf("player %q: Connected = %v, want false", p.Name, p.Connected)
		}
		switch p.Role {
		case Undercover:
			undercovers++
		case Blank:
			blanks++
		case Civilian:
			civilians++
		default:
			t.Errorf("player %q: unexpected role %v", p.Name, p.Role)
		}
	}
	if undercovers != 1 {
		t.Errorf("undercovers = %d, want 1", undercovers)
	}
	if blanks != 1 {
		t.Errorf("blanks = %d, want 1", blanks)
	}
	if civilians != 3 {
		t.Errorf("civilians = %d, want 3", civilians)
	}

	gotNames := make(map[string]bool)
	for _, p := range players {
		gotNames[p.Name] = true
	}
	for _, n := range names {
		if !gotNames[n] {
			t.Errorf("name %q not found in result", n)
		}
	}
}

func TestAssignRoles_FourPlayers(t *testing.T) {
	names := []string{"alice", "bob", "carol", "dave"}
	players, err := AssignRoles(names, 1, 0)
	if err != nil {
		t.Fatalf("AssignRoles() unexpected error: %v", err)
	}
	if len(players) != 4 {
		t.Fatalf("got %d players, want 4", len(players))
	}

	var undercovers, blanks, civilians int
	for _, p := range players {
		if p.Name == "" {
			t.Error("player has empty Name")
		}
		if !p.Alive {
			t.Errorf("player %q: Alive = false, want true", p.Name)
		}
		if p.Connected {
			t.Errorf("player %q: Connected = true, want false", p.Name)
		}
		switch p.Role {
		case Undercover:
			undercovers++
		case Blank:
			blanks++
		case Civilian:
			civilians++
		default:
			t.Errorf("player %q: unexpected role %v", p.Name, p.Role)
		}
	}
	if undercovers != 1 {
		t.Errorf("undercovers = %d, want 1", undercovers)
	}
	if blanks != 0 {
		t.Errorf("blanks = %d, want 0", blanks)
	}
	if civilians != 3 {
		t.Errorf("civilians = %d, want 3", civilians)
	}

	gotNames := make(map[string]bool)
	for _, p := range players {
		gotNames[p.Name] = true
	}
	for _, n := range names {
		if !gotNames[n] {
			t.Errorf("name %q not found in result", n)
		}
	}
}

func TestAssignRoles_TenPlayers(t *testing.T) {
	names := make([]string, 10)
	for i := range names {
		names[i] = fmt.Sprintf("player%d", i)
	}
	players, err := AssignRoles(names, 3, 1)
	if err != nil {
		t.Fatalf("AssignRoles() unexpected error: %v", err)
	}
	if len(players) != 10 {
		t.Fatalf("got %d players, want 10", len(players))
	}

	var undercovers, blanks, civilians int
	for _, p := range players {
		if p.Name == "" {
			t.Error("player has empty Name")
		}
		if !p.Alive {
			t.Errorf("player %q: Alive = false, want true", p.Name)
		}
		if p.Connected {
			t.Errorf("player %q: Connected = true, want false", p.Name)
		}
		switch p.Role {
		case Undercover:
			undercovers++
		case Blank:
			blanks++
		case Civilian:
			civilians++
		default:
			t.Errorf("player %q: unexpected role %v", p.Name, p.Role)
		}
	}
	if undercovers != 3 {
		t.Errorf("undercovers = %d, want 3", undercovers)
	}
	if blanks != 1 {
		t.Errorf("blanks = %d, want 1", blanks)
	}
	if civilians != 6 {
		t.Errorf("civilians = %d, want 6", civilians)
	}
}

func TestAssignRoles_AllCivilianBoundary(t *testing.T) {
	names := []string{"a", "b", "c", "d"}
	players, err := AssignRoles(names, 0, 0)
	if err != nil {
		t.Fatalf("AssignRoles() unexpected error: %v", err)
	}
	if len(players) != 4 {
		t.Fatalf("got %d players, want 4", len(players))
	}
	for _, p := range players {
		if p.Role != Civilian {
			t.Errorf("player %q: role = %v, want Civilian", p.Name, p.Role)
		}
		if !p.Alive {
			t.Errorf("player %q: Alive = false, want true", p.Name)
		}
		if p.Connected {
			t.Errorf("player %q: Connected = true, want false", p.Name)
		}
	}
}

func TestAssignRoles_Shuffled(t *testing.T) {
	names := make([]string, 20)
	for i := range names {
		names[i] = fmt.Sprintf("p%d", i)
	}

	sameOrder := true
	for trial := 0; trial < 5 && sameOrder; trial++ {
		players, err := AssignRoles(names, 2, 2)
		if err != nil {
			t.Fatalf("AssignRoles() unexpected error: %v", err)
		}
		sameOrder = true
		for i, p := range players {
			if p.Name != names[i] {
				sameOrder = false
				break
			}
		}
	}
	if sameOrder {
		t.Error("AssignRoles() returned names in original order across 5 trials; shuffle may not be working")
	}
}

func TestAssignRoles_NoUndercover(t *testing.T) {
	names := []string{"alice", "bob", "carol"}
	players, err := AssignRoles(names, 0, 0)
	if err != nil {
		t.Fatalf("AssignRoles() unexpected error: %v", err)
	}
	for _, p := range players {
		if p.Role != Civilian {
			t.Errorf("player %q: role = %v, want Civilian", p.Name, p.Role)
		}
	}
}

func TestAssignRoles_Errors(t *testing.T) {
	tests := []struct {
		name          string
		names         []string
		undercover    int
		blank         int
		wantErrSubstr string
	}{
		{"empty names", []string{}, 1, 0, "must not be empty"},
		{"nil names", nil, 1, 0, "must not be empty"},
		{"empty name in list", []string{"alice", "", "carol"}, 1, 0, "must not be empty"},
		{"negative undercover", []string{"a", "b", "c"}, -1, 0, "must not be negative"},
		{"negative blank", []string{"a", "b", "c"}, 0, -1, "must not be negative"},
		{"U+B equals C exactly", []string{"a", "b", "c", "d"}, 1, 1, "must be less than"},
		{"undercover+blank equals total", []string{"a", "b"}, 1, 1, "must be less than"},
		{"undercover+blank exceeds total", []string{"a", "b"}, 2, 1, "must be less than"},
		{"undercover+blank >= civilian", []string{"a", "b", "c", "d", "e", "f"}, 3, 1, "must be less than"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AssignRoles(tt.names, tt.undercover, tt.blank)
			if err == nil {
				t.Fatal("AssignRoles() expected error, got nil")
			}
			if tt.wantErrSubstr != "" && !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("AssignRoles() error %q does not contain %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}
