// Package gateway accepts WebSocket connections, performs the Hello handshake and routes messages.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/sim"
)

const (
	protocolVersion = uint32(pb.ProtocolVersion_PROTOCOL_VERSION_CURRENT)
	maxMessageBytes = 64 << 10
	writeTimeout    = 5 * time.Second
)

// Config configures the gateway handler.
type Config struct {
	Hub            *sim.Hub
	ServerName     string
	HelloTimeout   time.Duration
	OriginPatterns []string // extra allowed Origin hosts, e.g. the Vite dev server
	Logger         *slog.Logger
}

// Gateway is an http.Handler for the game WebSocket endpoint.
type Gateway struct{ cfg Config }

// New returns a Gateway with defaults applied.
func New(cfg Config) *Gateway {
	if cfg.HelloTimeout == 0 {
		cfg.HelloTimeout = 10 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Gateway{cfg: cfg}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: g.cfg.OriginPatterns})
	if err != nil {
		g.cfg.Logger.Debug("websocket accept failed", "err", err)
		return
	}
	c.SetReadLimit(maxMessageBytes)
	sess := &session{gw: g, conn: c, log: g.cfg.Logger.With("remote", r.RemoteAddr)}
	err = sess.run(r.Context())
	sess.close(err)
}

type session struct {
	gw   *Gateway
	conn *websocket.Conn
	log  *slog.Logger
}

// rejectError carries a reason that is sent to the client before closing.
type rejectError struct{ reason string }

func (e rejectError) Error() string { return e.reason }

func (s *session) run(ctx context.Context) error {
	if err := s.handshake(ctx); err != nil {
		return err
	}
	hub := s.gw.cfg.Hub
	hub.Join()
	s.log.Info("player joined", "players", hub.Players())
	defer func() {
		hub.Leave()
		s.log.Info("player left", "players", hub.Players(), "idle", !hub.Running())
	}()

	for {
		msg, err := s.read(ctx)
		if err != nil {
			return err
		}
		if err := s.handle(ctx, msg); err != nil {
			return err
		}
	}
}

func (s *session) handshake(ctx context.Context) error {
	hctx, cancel := context.WithTimeout(ctx, s.gw.cfg.HelloTimeout)
	defer cancel()
	msg, err := s.read(hctx)
	if err != nil {
		return err
	}
	hello := msg.GetHello()
	if hello == nil {
		return rejectError{"expected hello"}
	}
	if hello.ProtocolVersion != protocolVersion {
		return rejectError{fmt.Sprintf("protocol version %d unsupported, server is %d", hello.ProtocolVersion, protocolVersion)}
	}
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_Welcome{Welcome: &pb.Welcome{
		ProtocolVersion: protocolVersion,
		ServerName:      s.gw.cfg.ServerName,
		TickRateHz:      uint32(s.gw.cfg.Hub.TickRate()),
	}}})
}

func (s *session) handle(ctx context.Context, msg *pb.ClientMessage) error {
	switch body := msg.Body.(type) {
	case *pb.ClientMessage_Ping:
		return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_Pong{Pong: &pb.Pong{
			ClientTimeMs: body.Ping.ClientTimeMs,
			ServerTimeMs: float64(time.Now().UnixMicro()) / 1000,
		}}})
	case *pb.ClientMessage_Hello:
		return rejectError{"duplicate hello"}
	default:
		return rejectError{"unknown message"}
	}
}

func (s *session) read(ctx context.Context) (*pb.ClientMessage, error) {
	typ, b, err := s.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	if typ != websocket.MessageBinary {
		return nil, rejectError{"binary messages only"}
	}
	var msg pb.ClientMessage
	if err := proto.Unmarshal(b, &msg); err != nil {
		return nil, rejectError{"malformed message"}
	}
	return &msg, nil
}

func (s *session) write(ctx context.Context, msg *pb.ServerMessage) error {
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	wctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return s.conn.Write(wctx, websocket.MessageBinary, b)
}

func (s *session) close(err error) {
	var rej rejectError
	if errors.As(err, &rej) {
		s.log.Info("rejecting client", "reason", rej.reason)
		_ = s.write(context.Background(), &pb.ServerMessage{Body: &pb.ServerMessage_Disconnect{Disconnect: &pb.Disconnect{Reason: rej.reason}}})
		_ = s.conn.Close(websocket.StatusPolicyViolation, rej.reason)
		return
	}
	if websocket.CloseStatus(err) == -1 && err != nil {
		s.log.Debug("connection ended", "err", err)
	}
	_ = s.conn.CloseNow()
}
