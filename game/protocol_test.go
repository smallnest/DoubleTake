package game

import (
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
		want string
	}{
		{"with payload", Message{Type: MsgJoin, Payload: "alice"}, "JOIN|alice\n"},
		{"empty payload", Message{Type: MsgReady, Payload: ""}, "READY|\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Encode(tt.msg); got != tt.want {
				t.Errorf("Encode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Message
		wantErr bool
	}{
		{"valid message", "JOIN|alice\n", Message{Type: MsgJoin, Payload: "alice"}, false},
		{"pipe in payload", "DESC|这是含|分隔的内容\n", Message{Type: MsgDesc, Payload: "这是含|分隔的内容"}, false},
		{"empty payload", "READY|\n", Message{Type: MsgReady, Payload: ""}, false},
		{"empty string", "", Message{}, true},
		{"newline only", "\n", Message{}, true},
		{"no separator", "JOIN", Message{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Decode() = %v, want %v", got, tt.want)
			}
		})
	}
}
