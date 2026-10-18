package gateway

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// clientIP returns the caller's IP. X-Forwarded-For is trusted only from loopback or private peers
// (e.g. Caddy on the Docker network), using its rightmost entry, the one the proxy added.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil || !trustedProxy(peer) {
		return host
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return host
	}
	hops := strings.Split(xff, ",")
	return strings.TrimSpace(hops[len(hops)-1])
}

func trustedProxy(peer netip.Addr) bool { return peer.IsLoopback() || peer.IsPrivate() }
