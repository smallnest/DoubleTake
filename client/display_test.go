package client

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestPrintStartup(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, false)
	d.PrintStartup()

	got := strings.TrimSpace(buf.String())
	want := "[sysmon] v2.4.1 — diagnostics mode"
	if got != want {
		t.Errorf("PrintStartup() = %q, want %q", got, want)
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, false)
	d.Info("7a3f", "node registered")

	got := strings.TrimSpace(buf.String())
	want := "[INFO] [session:7a3f] node registered"
	if got != want {
		t.Errorf("Info() = %q, want %q", got, want)
	}
}

func TestInfoStealth(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, true)
	d.Info("7a3f", "node registered")

	got := strings.TrimSpace(buf.String())
	want := "[session:7a3f] node registered"
	if got != want {
		t.Errorf("Info() stealth = %q, want %q", got, want)
	}
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, false)
	d.Warn("connection unstable")

	got := strings.TrimSpace(buf.String())
	want := "[WARN] connection unstable"
	if got != want {
		t.Errorf("Warn() = %q, want %q", got, want)
	}
}

func TestWarnStealth(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, true)
	d.Warn("connection unstable")

	got := strings.TrimSpace(buf.String())
	want := "warning: connection unstable"
	if got != want {
		t.Errorf("Warn() stealth = %q, want %q", got, want)
	}
}

func TestData(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, false)
	d.Data("05", "heartbeat received")

	got := strings.TrimSpace(buf.String())
	want := "[DATA] [node-05] heartbeat received"
	if got != want {
		t.Errorf("Data() = %q, want %q", got, want)
	}
}

func TestDataStealth(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, true)
	d.Data("05", "heartbeat received")

	got := strings.TrimSpace(buf.String())
	want := "[node-05] heartbeat received"
	if got != want {
		t.Errorf("Data() stealth = %q, want %q", got, want)
	}
}

func TestListPlayers(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, false)
	d.ListPlayers([]string{"alice", "bob", "carol"})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("ListPlayers() returned %d lines, want 3", len(lines))
	}

	tests := []struct {
		idx    int
		player string
	}{
		{0, "alice"},
		{1, "bob"},
		{2, "carol"},
	}
	for _, tt := range tests {
		want := fmt.Sprintf("connections.list[%d]: %s  status=active", tt.idx, tt.player)
		if lines[tt.idx] != want {
			t.Errorf("ListPlayers()[%d] = %q, want %q", tt.idx, lines[tt.idx], want)
		}
	}
}

func TestListPlayersEmpty(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(&buf, false)
	d.ListPlayers(nil)

	if buf.Len() != 0 {
		t.Errorf("ListPlayers(nil) = %q, want empty", buf.String())
	}
}

func TestNewDisplayNilWriter(t *testing.T) {
	d := NewDisplay(nil, false)
	if d == nil {
		t.Fatal("NewDisplay(nil, false) returned nil")
	}
}
