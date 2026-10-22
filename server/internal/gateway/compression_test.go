package gateway

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestBrowsersGetCompressedMessages(t *testing.T) {
	ts := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c, resp, err := websocket.Dial(ctx, ts.url, &websocket.DialOptions{CompressionMode: websocket.CompressionContextTakeover})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.CloseNow() })
	ext := resp.Header.Get("Sec-WebSocket-Extensions")
	if !strings.Contains(ext, "permessage-deflate") || !strings.Contains(ext, "server_no_context_takeover") {
		t.Fatalf("extensions %q: want per-message deflate without kept context", ext)
	}
}
