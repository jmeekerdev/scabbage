package gateway

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"scabbage/internal/access"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/rate"
)

const (
	maxMessageBytes = 64 << 10
	writeTimeout    = 5 * time.Second
	outQueueSize    = 64
)

// session is one WebSocket connection. Only its run goroutine writes to the socket.
type session struct {
	gw      *Gateway
	conn    *websocket.Conn
	ip      string
	log     *slog.Logger
	in      chan inbound
	out     chan *pb.ServerMessage
	player  access.Player
	console bool   // admin console: no world, no hub
	skin    []byte // approved skin, nil for the default; guarded by gw.mu once in the world

	body     *motion.Tracker // the player's server-side position; nil for the console
	teleport uint32          // id of the last Teleport sent
	edits    *rate.Bucket
	talk     *rate.Bucket // chat and emotes

	cancel context.CancelFunc // ends the session, e.g. when it can't keep up
}

type inbound struct {
	msg *pb.ClientMessage
	err error
}

// rejectError carries a reason that is sent to the client before closing.
type rejectError struct {
	code   pb.DisconnectCode
	reason string
}

func (e rejectError) Error() string { return e.reason }

func reject(code pb.DisconnectCode, reason string) rejectError { return rejectError{code, reason} }

func newSession(g *Gateway, c *websocket.Conn, ip string, log *slog.Logger, cancel context.CancelFunc) *session {
	return &session{gw: g, conn: c, ip: ip, log: log, cancel: cancel, in: make(chan inbound), out: make(chan *pb.ServerMessage, outQueueSize)}
}

func (s *session) run(ctx context.Context) error {
	hello, err := s.readHello(ctx)
	if err != nil {
		return err
	}
	s.console = hello.Console
	go s.readLoop(ctx)
	if err := s.admit(ctx, hello); err != nil {
		return err
	}
	if !s.console {
		hub := s.gw.cfg.Hub
		hub.Join()
		s.gw.addPlayer(s)
		s.log.Info("player joined", "name", s.player.Name, "players", hub.Players())
		defer func() {
			s.gw.removePlayer(s)
			hub.Leave()
			s.log.Info("player left", "name", s.player.Name, "players", hub.Players(), "idle", !hub.Running())
		}()
	}
	if s.player.Admin {
		s.gw.addAdmin(s)
		defer s.gw.removeAdmin(s)
		if err := s.sendPending(ctx); err != nil {
			return err
		}
		if err := s.sendArtPending(ctx); err != nil {
			return err
		}
	}
	return s.serve(ctx)
}

// serve handles client messages and queued pushes until the connection ends.
func (s *session) serve(ctx context.Context) error {
	for {
		select {
		case in := <-s.in:
			if in.err != nil {
				return in.err
			}
			if err := s.handle(ctx, in.msg); err != nil {
				return err
			}
		case msg := <-s.out:
			if err := s.write(ctx, msg); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *session) handle(ctx context.Context, msg *pb.ClientMessage) error {
	switch body := msg.Body.(type) {
	case *pb.ClientMessage_Ping:
		return s.pong(ctx, body.Ping)
	case *pb.ClientMessage_DecideJoin:
		return s.decideJoin(ctx, body.DecideJoin)
	case *pb.ClientMessage_CreateJoinCode:
		return s.createJoinCode(ctx, body.CreateJoinCode)
	case *pb.ClientMessage_RequestChunks:
		return s.sendChunks(ctx, body.RequestChunks)
	case *pb.ClientMessage_Move:
		return s.move(ctx, body.Move)
	case *pb.ClientMessage_EditBlock:
		return s.editBlock(ctx, body.EditBlock)
	case *pb.ClientMessage_SendChat:
		return s.sendChat(ctx, body.SendChat)
	case *pb.ClientMessage_SendEmote:
		return s.sendEmote(body.SendEmote)
	case *pb.ClientMessage_SubmitArt:
		return s.submitArt(ctx, body.SubmitArt)
	case *pb.ClientMessage_DecideArt:
		return s.decideArt(ctx, body.DecideArt)
	case *pb.ClientMessage_Hello:
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "duplicate hello")
	default:
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "unknown message")
	}
}

func (s *session) pong(ctx context.Context, p *pb.Ping) error {
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_Pong{Pong: &pb.Pong{
		ClientTimeMs: p.ClientTimeMs,
		ServerTimeMs: float64(time.Now().UnixMicro()) / 1000,
	}}})
}

// push queues a message from another goroutine. A client too far behind is disconnected rather
// than silently missing world changes; it resyncs when it reconnects.
func (s *session) push(msg *pb.ServerMessage) {
	select {
	case s.out <- msg:
	default:
		s.log.Warn("disconnecting slow client")
		s.cancel()
	}
}

func (s *session) readLoop(ctx context.Context) {
	for {
		msg, err := s.read(ctx)
		select {
		case s.in <- inbound{msg, err}:
		case <-ctx.Done():
			return
		}
		if err != nil {
			return
		}
	}
}

func (s *session) read(ctx context.Context) (*pb.ClientMessage, error) {
	typ, b, err := s.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	if typ != websocket.MessageBinary {
		return nil, reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "binary messages only")
	}
	var msg pb.ClientMessage
	if err := proto.Unmarshal(b, &msg); err != nil {
		return nil, reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "malformed message")
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
		s.log.Info("rejecting client", "code", rej.code, "reason", rej.reason)
		_ = s.write(context.Background(), &pb.ServerMessage{Body: &pb.ServerMessage_Disconnect{
			Disconnect: &pb.Disconnect{Code: rej.code, Reason: rej.reason},
		}})
		_ = s.conn.Close(websocket.StatusPolicyViolation, rej.reason)
		return
	}
	if err != nil && websocket.CloseStatus(err) == -1 && !errors.Is(err, context.Canceled) {
		s.log.Debug("connection ended", "err", err)
	}
	_ = s.conn.CloseNow()
}
