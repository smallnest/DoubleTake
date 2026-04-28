package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_ValidRoleJudge(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--role", "judge"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Starting DoubleTake in judge mode on port 8127") {
		t.Errorf("expected judge mode message, got: %s", stdout.String())
	}
}

func TestRun_ValidRolePlayer(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--role", "player"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Starting DoubleTake in player mode on port 8127") {
		t.Errorf("expected player mode message, got: %s", stdout.String())
	}
}

func TestRun_CustomPort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--role", "judge", "--port", "9000"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "port 9000") {
		t.Errorf("expected port 9000 in output, got: %s", stdout.String())
	}
}

func TestRun_DefaultPort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--role", "player"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "port 8127") {
		t.Errorf("expected default port 8127, got: %s", stdout.String())
	}
}

func TestRun_InvalidRole(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--role", "admin"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "invalid role 'admin'") {
		t.Errorf("expected invalid role error, got: %s", stderr.String())
	}
}

func TestRun_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage output, got: %s", stdout.String())
	}
}

func TestRun_HelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--help"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage output, got: %s", stdout.String())
	}
}

func TestRun_ShortHelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "-h"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage output, got: %s", stdout.String())
	}
}

func TestRun_MissingRoleValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--role"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "--role requires a value") {
		t.Errorf("expected missing role value error, got: %s", stderr.String())
	}
}

func TestRun_MissingPortValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--role", "judge", "--port"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "--port requires a value") {
		t.Errorf("expected missing port value error, got: %s", stderr.String())
	}
}

func TestRun_InvalidPort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--role", "judge", "--port", "abc"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "invalid port value") {
		t.Errorf("expected invalid port error, got: %s", stderr.String())
	}
}

func TestRun_UnknownOption(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(&stdout, &stderr, []string{"doubletake", "--unknown"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown option") {
		t.Errorf("expected unknown option error, got: %s", stderr.String())
	}
}
