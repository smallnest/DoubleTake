package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestValidateConfig_Valid(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		undercovers int
		blanks      int
	}{
		{"minimum 4 players 1 undercover", 4, 1, 0},
		{"5 players 2 undercovers", 5, 2, 0},
		{"6 players 1 undercover 1 blank", 6, 1, 1},
		{"8 players 2 undercovers 1 blank", 8, 2, 1},
		{"10 players 3 undercovers", 10, 3, 0},
		{"10 players 2 undercovers 2 blanks", 10, 2, 2},
		{"7 players 1 undercover 2 blanks", 7, 1, 2},
		{"9 players 2 undercovers 1 blank", 9, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.total, tt.undercovers, tt.blanks)
			if err != nil {
				t.Errorf("validateConfig(%d, %d, %d) returned error: %v", tt.total, tt.undercovers, tt.blanks, err)
			}
		})
	}
}

func TestValidateConfig_TotalOutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		total int
	}{
		{"too few 3", 3},
		{"too few 1", 1},
		{"too few 0", 0},
		{"too many 11", 11},
		{"too many 15", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.total, 1, 0)
			if err == nil {
				t.Errorf("expected error for total=%d", tt.total)
			}
			if !strings.Contains(err.Error(), "玩家人数") {
				t.Errorf("error should mention 玩家人数, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_UndercoverOutOfRange(t *testing.T) {
	tests := []struct {
		name        string
		undercovers int
	}{
		{"zero undercovers", 0},
		{"negative undercovers", -1},
		{"too many undercovers 4", 4},
		{"too many undercovers 5", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(6, tt.undercovers, 0)
			if err == nil {
				t.Errorf("expected error for undercovers=%d", tt.undercovers)
			}
			if !strings.Contains(err.Error(), "卧底人数") {
				t.Errorf("error should mention 卧底人数, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_NegativeBlanks(t *testing.T) {
	err := validateConfig(6, 1, -1)
	if err == nil {
		t.Error("expected error for negative blanks")
	}
	if !strings.Contains(err.Error(), "白板人数") {
		t.Errorf("error should mention 白板人数, got: %v", err)
	}
}

func TestValidateConfig_TooManySpecialRoles(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		undercovers int
		blanks      int
	}{
		{"4 players 2 undercovers", 4, 2, 0},
		{"5 players 2 undercovers 1 blank", 5, 2, 1},
		{"6 players 3 undercovers", 6, 3, 0},
		{"8 players 3 undercovers 2 blanks", 8, 3, 2},
		{"10 players 3 undercovers 3 blanks", 10, 3, 3},
		{"4 players 1 undercover 1 blank", 4, 1, 1},
		{"6 players 2 undercovers 1 blank", 6, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.total, tt.undercovers, tt.blanks)
			if err == nil {
				t.Errorf("expected error for total=%d undercovers=%d blanks=%d", tt.total, tt.undercovers, tt.blanks)
			}
			if !strings.Contains(err.Error(), "卧底") || !strings.Contains(err.Error(), "白板") {
				t.Errorf("error should mention 卧底 and 白板, got: %v", err)
			}
		})
	}
}

func TestCollectConfig_ValidInput(t *testing.T) {
	input := "6\n1\n1\n"
	out := &bytes.Buffer{}
	cfg := runJudgeWithInput(out, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
	if cfg.Undercovers != 1 {
		t.Errorf("expected undercovers 1, got %d", cfg.Undercovers)
	}
	if cfg.Blanks != 1 {
		t.Errorf("expected blanks 1, got %d", cfg.Blanks)
	}
}

func TestCollectConfig_InvalidTotalThenValid(t *testing.T) {
	input := "3\n1\n0\n6\n1\n0\n"
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
	if cfg.Undercovers != 1 {
		t.Errorf("expected undercovers 1, got %d", cfg.Undercovers)
	}
	if cfg.Blanks != 0 {
		t.Errorf("expected blanks 0, got %d", cfg.Blanks)
	}
}

func TestCollectConfig_InvalidUndercoverThenValid(t *testing.T) {
	input := "6\n4\n0\n6\n1\n0\n"
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_NegativeBlanksThenValid(t *testing.T) {
	input := "6\n1\n-1\n6\n1\n0\n"
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.Blanks != 0 {
		t.Errorf("expected blanks 0, got %d", cfg.Blanks)
	}
}

func TestCollectConfig_TooManySpecialRolesThenValid(t *testing.T) {
	input := "4\n1\n1\n6\n1\n0\n"
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_MultipleRetries(t *testing.T) {
	input := "3\n1\n0\n" + // total too low
		"6\n0\n0\n" + // undercover too low
		"6\n1\n-1\n" + // blanks negative
		"6\n1\n0\n" // valid
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
	if cfg.Undercovers != 1 {
		t.Errorf("expected undercovers 1, got %d", cfg.Undercovers)
	}
	if cfg.Blanks != 0 {
		t.Errorf("expected blanks 0, got %d", cfg.Blanks)
	}
}

func TestCollectConfig_BoundaryMaxPlayers(t *testing.T) {
	input := "10\n1\n0\n"
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 10 {
		t.Errorf("expected total 10, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_BoundaryMinPlayers(t *testing.T) {
	input := "4\n1\n0\n"
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 4 {
		t.Errorf("expected total 4, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_WarnsOnInvalidTotal(t *testing.T) {
	out := &bytes.Buffer{}
	input := "3\n1\n0\n6\n1\n0\n"
	runJudgeWithInput(out, input)

	output := out.String()
	if !strings.Contains(output, "玩家人数 3 不在合法范围") {
		t.Errorf("expected warning about total players, got: %s", output)
	}
}

func TestCollectConfig_WarnsOnInvalidUndercover(t *testing.T) {
	out := &bytes.Buffer{}
	input := "6\n0\n0\n6\n1\n0\n"
	runJudgeWithInput(out, input)

	output := out.String()
	if !strings.Contains(output, "卧底人数 0 不在合法范围") {
		t.Errorf("expected warning about undercover count, got: %s", output)
	}
}

func TestCollectConfig_WarnsOnNegativeBlanks(t *testing.T) {
	out := &bytes.Buffer{}
	input := "6\n1\n-1\n6\n1\n0\n"
	runJudgeWithInput(out, input)

	output := out.String()
	if !strings.Contains(output, "白板人数 -1 不能为负数") {
		t.Errorf("expected warning about negative blanks, got: %s", output)
	}
}

func TestCollectConfig_WarnsOnTooManySpecialRoles(t *testing.T) {
	out := &bytes.Buffer{}
	input := "4\n1\n1\n6\n1\n0\n"
	runJudgeWithInput(out, input)

	output := out.String()
	if !strings.Contains(output, "卧底(1)+白板(1)=2，必须少于总人数的一半(2)") {
		t.Errorf("expected warning about special roles ratio, got: %s", output)
	}
}

func TestReadInt_ValidInput(t *testing.T) {
	scanner := newTestScanner("42\n")
	n, err := readInt(scanner)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
}

func TestReadInt_TrimsWhitespace(t *testing.T) {
	scanner := newTestScanner("  7  \n")
	n, err := readInt(scanner)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 7 {
		t.Errorf("expected 7, got %d", n)
	}
}

func TestReadInt_InvalidInput(t *testing.T) {
	scanner := newTestScanner("abc\n")
	_, err := readInt(scanner)
	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}

func TestReadInt_EmptyInput(t *testing.T) {
	scanner := newTestScanner("")
	_, err := readInt(scanner)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestCollectConfig_NonNumericInput(t *testing.T) {
	// First iteration: total=abc → -1, undercover=4, blanks=4 -> validation fails (total<4)
	// Second iteration: total=6, undercover=1, blanks=0 -> valid
	input := "abc\n4\n4\n6\n1\n0\n"
	out := &bytes.Buffer{}
	cfg := runJudgeWithInput(out, input)

	if cfg.TotalPlayers != 6 {
		t.Errorf("expected total 6, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_ZeroBlanksAllowed(t *testing.T) {
	input := "4\n1\n0\n"
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.Blanks != 0 {
		t.Errorf("expected blanks 0, got %d", cfg.Blanks)
	}
	if cfg.TotalPlayers != 4 {
		t.Errorf("expected total 4, got %d", cfg.TotalPlayers)
	}
}

func TestCollectConfig_LargeValidGame(t *testing.T) {
	input := "10\n3\n1\n"
	cfg := runJudgeWithInput(&bytes.Buffer{}, input)

	if cfg.TotalPlayers != 10 {
		t.Errorf("expected total 10, got %d", cfg.TotalPlayers)
	}
	if cfg.Undercovers != 3 {
		t.Errorf("expected undercovers 3, got %d", cfg.Undercovers)
	}
	if cfg.Blanks != 1 {
		t.Errorf("expected blanks 1, got %d", cfg.Blanks)
	}
}

// newTestScanner creates a bufio.Scanner from a string for testing.
func newTestScanner(input string) *bufio.Scanner {
	return bufio.NewScanner(strings.NewReader(input))
}

// runJudgeWithInput is a test helper that runs RunJudge with the given input string.
func runJudgeWithInput(out *bytes.Buffer, input string) GameConfig {
	return RunJudge(out, strings.NewReader(input), "8127", false)
}
