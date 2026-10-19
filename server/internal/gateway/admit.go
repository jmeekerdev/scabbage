package gateway

import (
	"context"
	"errors"
	"fmt"
	"time"

	"scabbage/internal/access"
	"scabbage/internal/chat"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/rate"
)

const protocolVersion = uint32(pb.ProtocolVersion_PROTOCOL_VERSION_CURRENT)

func (s *session) readHello(ctx context.Context) (*pb.Hello, error) {
	hctx, cancel := context.WithTimeout(ctx, s.gw.cfg.HelloTimeout)
	defer cancel()
	msg, err := s.read(hctx)
	if err != nil {
		return nil, err
	}
	hello := msg.GetHello()
	if hello == nil {
		return nil, reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "expected hello")
	}
	if hello.ProtocolVersion != protocolVersion {
		return nil, reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL,
			fmt.Sprintf("protocol version %d unsupported, server is %d (reload the page)", hello.ProtocolVersion, protocolVersion))
	}
	return hello, nil
}

// admit resolves the device to a player, waiting for an admin if needed, then sends Welcome.
func (s *session) admit(ctx context.Context, hello *pb.Hello) error {
	res, err := s.authenticate(hello)
	if err != nil {
		return rejectFor(err)
	}
	if res.Status == access.StatusPending {
		if res, err = s.awaitApproval(ctx, hello.DeviceSecret, res.RequestID); err != nil {
			return err
		}
	}
	s.player = res.Player
	if hello.Console && !s.player.Admin {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_NOT_ADMIN, "the admin console is for admins")
	}
	welcome := &pb.Welcome{
		ProtocolVersion: protocolVersion,
		ServerName:      s.gw.cfg.ServerName,
		TickRateHz:      uint32(s.gw.cfg.Hub.TickRate()),
		You:             playerInfo(s.player),
	}
	if !s.console {
		welcome.World = worldInfo(s.gw.cfg.World)
		s.startMotion()
		s.talk = rate.New(time.Now(), chat.PerSecond, chat.Burst)
	}
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_Welcome{Welcome: welcome}})
}

func (s *session) authenticate(hello *pb.Hello) (access.Result, error) {
	svc := s.gw.cfg.Access
	res, err := svc.Check(hello.DeviceSecret)
	if errors.Is(err, access.ErrUnknownDevice) && hello.Join != nil {
		j := hello.Join
		return svc.RequestJoin(hello.DeviceSecret, access.Join{Code: j.Code, Name: j.Name, Color: j.Color}, s.ip)
	}
	return res, err
}

// awaitApproval keeps a pending connection alive (answering pings) until an admin decides.
func (s *session) awaitApproval(ctx context.Context, secret []byte, requestID int64) (access.Result, error) {
	decision, cancel := s.gw.cfg.Access.Await(requestID)
	defer cancel()
	if err := s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_JoinPending{JoinPending: &pb.JoinPending{}}}); err != nil {
		return access.Result{}, err
	}
	s.log.Info("join request waiting", "request", requestID)
	for {
		select {
		case approved := <-decision:
			if !approved {
				return access.Result{}, rejectFor(access.ErrDenied)
			}
			res, err := s.gw.cfg.Access.Check(secret)
			return res, rejectFor(err)
		case in := <-s.in:
			if in.err != nil {
				return access.Result{}, in.err
			}
			if in.msg.GetPing() == nil {
				return access.Result{}, reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "waiting for approval")
			}
			if err := s.pong(ctx, in.msg.GetPing()); err != nil {
				return access.Result{}, err
			}
		case <-ctx.Done():
			return access.Result{}, ctx.Err()
		}
	}
}

// rejectFor turns access errors into client-visible disconnects; other errors pass through.
func rejectFor(err error) error {
	codes := []struct {
		err  error
		code pb.DisconnectCode
	}{
		{access.ErrBadSecret, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL},
		{access.ErrUnknownDevice, pb.DisconnectCode_DISCONNECT_CODE_NEED_JOIN_CODE},
		{access.ErrBadCode, pb.DisconnectCode_DISCONNECT_CODE_BAD_JOIN_CODE},
		{access.ErrBadName, pb.DisconnectCode_DISCONNECT_CODE_BAD_NAME},
		{access.ErrDenied, pb.DisconnectCode_DISCONNECT_CODE_DENIED},
		{access.ErrRateLimited, pb.DisconnectCode_DISCONNECT_CODE_RATE_LIMITED},
	}
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return reject(c.code, c.err.Error())
		}
	}
	return err
}
