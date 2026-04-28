package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_ValidRoleJudge(t *testing.T) {
	var stdout, stderr bytes.Buffer
	input := "6\n1\n0\n" // valid game config
	exitCode := run(&stdout, &stderr, strings.NewReader(input), []string{"doubletake", "--role", "judge"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "[sysmon]") {
		t.Errorf("expected startup banner, got: %s", stdout.String())
	}
}

func TestRun_ValidRolePlayer(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Empty stdin — RunPlayer fails immediately on room code prompt
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--role", "player"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 with no input, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "[sysmon]") {
		t.Errorf("expected startup banner, got: %s", stdout.String())
	}
}

func TestRun_CustomPort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	input := "6\n1\n0\n"
	exitCode := run(&stdout, &stderr, strings.NewReader(input), []string{"doubletake", "--role", "judge", "--port", "9000"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRun_DefaultPort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Player with no input → exit 1 (RunPlayer fails on empty stdin)
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--role", "player"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_InvalidRole(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--role", "admin"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "invalid role 'admin'") {
		t.Errorf("expected invalid role error, got: %s", stderr.String())
	}
}

func TestRun_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage output, got: %s", stdout.String())
	}
}

func TestRun_HelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--help"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage output, got: %s", stdout.String())
	}
}

func TestRun_ShortHelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "-h"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage output, got: %s", stdout.String())
	}
}

func TestRun_MissingRoleValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--role"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "--role requires a value") {
		t.Errorf("expected missing role value error, got: %s", stderr.String())
	}
}

func TestRun_MissingPortValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--role", "judge", "--port"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "--port requires a value") {
		t.Errorf("expected missing port value error, got: %s", stderr.String())
	}
}

func TestRun_InvalidPort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--role", "judge", "--port", "abc"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "invalid port value") {
		t.Errorf("expected invalid port error, got: %s", stderr.String())
	}
}

func TestRun_UnknownOption(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--unknown"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown option") {
		t.Errorf("expected unknown option error, got: %s", stderr.String())
	}
}

func TestRun_StealthFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Player with stealth — RunPlayer fails on empty stdin but output should lack [INFO]
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--role", "player", "--stealth"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	output := stdout.String()
	if strings.Contains(output, "[INFO]") {
		t.Errorf("stealth mode should not contain [INFO], got: %s", output)
	}
}

func TestRun_NoStealthByDefault(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Player without stealth should contain [INFO]
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--role", "player"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	output := stdout.String()
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("default mode should contain [INFO], got: %s", output)
	}
}

func TestRun_StealthInHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, strings.NewReader(""), []string{"doubletake", "--help"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "--stealth") {
		t.Errorf("help output should mention --stealth, got: %s", stdout.String())
	}
}

func TestRun_StealthWithJudge(t *testing.T) {
	var stdout, stderr bytes.Buffer
	input := "6\n1\n0\n"
	exitCode := run(&stdout, &stderr, strings.NewReader(input), []string{"doubletake", "--role", "judge", "--stealth"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := stdout.String()
	if strings.Contains(output, "[INFO]") {
		t.Errorf("stealth mode should not contain [INFO], got: %s", output)
	}
}
