package gateway

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name, remote, xff, want string
	}{
		{"direct public", "8.8.8.8:1234", "", "8.8.8.8"},
		{"public peer cannot spoof", "8.8.8.8:1234", "1.1.1.1", "8.8.8.8"},
		{"private proxy forwards", "172.18.0.3:5555", "203.0.113.9", "203.0.113.9"},
		{"rightmost hop wins", "127.0.0.1:5555", "6.6.6.6, 203.0.113.9", "203.0.113.9"},
		{"private without header", "192.168.1.20:5555", "", "192.168.1.20"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/ws", nil)
			r.RemoteAddr = tt.remote
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := clientIP(r); got != tt.want {
				t.Fatalf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
