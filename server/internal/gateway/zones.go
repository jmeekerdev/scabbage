package gateway

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"scabbage/internal/access"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/zones"
)

const (
	maxZoneSize = 256
	zoneHelp    = "Zones: /zone add NAME SIZE, /zone remove NAME, /zone allow NAME PLAYER, " +
		"/zone disallow NAME PLAYER, /zone mode NAME creative|survival|any, /zone list"
)

func wireMode(m access.Mode) pb.GameMode {
	switch m {
	case access.ModeCreative:
		return creative
	case access.ModeSurvival:
		return survival
	}
	return pb.GameMode_GAME_MODE_UNSPECIFIED
}

// zonesMessage is every zone as s sees it, with its arena; nil when there are none and skipEmpty is set.
func (g *Gateway) zonesMessage(s *session, skipEmpty bool) *pb.ServerMessage {
	all := g.cfg.Zones.All()
	if len(all) == 0 && skipEmpty {
		return nil
	}
	out := &pb.ZonesChanged{Zones: make([]*pb.Zone, 0, len(all))}
	for _, z := range all {
		arena := g.arenas[strings.ToLower(z.Name)]
		out.Zones = append(out.Zones, &pb.Zone{
			Name: z.Name, X0: z.X0, Z0: z.Z0, X1: z.X1, Z1: z.Z1,
			Build: s.player.Admin || z.Lets(s.player.ID), Mode: wireMode(z.Mode),
			Marks: arena.marks, Pvp: arena.arena.PVP,
		})
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_ZonesChanged{ZonesChanged: out}}
}

// zonesChanged sends everyone the zones and puts each player in the mode where they stand.
func (g *Gateway) zonesChanged() {
	arenas := g.arenaViews()
	g.mu.Lock()
	defer g.mu.Unlock()
	g.arenas = arenas
	for s, a := range g.players {
		s.push(g.zonesMessage(s, false))
		g.refreshModeLocked(s, a, false)
	}
}

// zoneMode is the mode a player plays where they stand: the zone's if it sets one, else their own.
func (g *Gateway) zoneMode(a *avatar) pb.GameMode {
	z, ok := g.cfg.Zones.At(int32(math.Floor(a.pos.X)), int32(math.Floor(a.pos.Z)))
	if ok && z.Mode != access.ModeDefault {
		return wireMode(z.Mode)
	}
	return a.own
}

// refreshModeLocked starts a player afresh when their mode changes, or always when forced.
func (g *Gateway) refreshModeLocked(s *session, a *avatar, force bool) {
	m := g.zoneMode(a)
	if m == a.mode && !force {
		return
	}
	a.mode = m
	a.vitals.Revive(time.Now())
	s.push(vitalsMessage(a, pb.Harm_HARM_UNSPECIFIED))
}

// zoneCommand runs an admin's /zone command.
func (s *session) zoneCommand(ctx context.Context, arg string) error {
	f := strings.Fields(arg)
	if len(f) == 0 || f[0] == "list" {
		return s.listZones(ctx)
	}
	args := 3
	if f[0] == "remove" {
		args = 2
	}
	if len(f) != args {
		return s.notice(ctx, zoneHelp)
	}
	var err error
	switch f[0] {
	case "add":
		err = s.addZone(f[1], f[2])
	case "remove":
		err = s.gw.cfg.Zones.Remove(f[1])
	case "allow", "disallow":
		err = s.allowInZone(f[1], f[2], f[0] == "allow")
	case "mode":
		err = s.setZoneMode(f[1], f[2])
	default:
		return s.notice(ctx, zoneHelp)
	}
	return s.zoneChanged(ctx, f, err)
}

// zoneChanged tells the admin how a /zone command went and everyone about the new zones.
func (s *session) zoneChanged(ctx context.Context, f []string, err error) error {
	var bad badZoneArg
	switch {
	case errors.As(err, &bad):
		return s.notice(ctx, string(bad))
	case errors.Is(err, zones.ErrNoZone):
		return s.notice(ctx, fmt.Sprintf("No zone is called %q.", f[1]))
	case errors.Is(err, zones.ErrBadName):
		return s.notice(ctx, "Zone names are 1 to 24 letters.")
	case err != nil:
		return err
	}
	s.log.Info("zone changed", "command", strings.Join(f, " "), "by", s.player.Name)
	s.gw.zonesChanged()
	return s.notice(ctx, "Zones updated.")
}

// badZoneArg is a /zone mistake to explain to the admin.
type badZoneArg string

func (e badZoneArg) Error() string { return string(e) }

// addZone makes a square zone of the given size centred on the admin.
func (s *session) addZone(name, size string) error {
	n, err := strconv.Atoi(size)
	if err != nil || n < 1 || n > maxZoneSize {
		return badZoneArg(fmt.Sprintf("SIZE is a number of blocks from 1 to %d.", maxZoneSize))
	}
	at := s.body.Pos()
	x0, z0 := int32(math.Floor(at.X))-int32(n/2), int32(math.Floor(at.Z))-int32(n/2)
	return s.gw.cfg.Zones.Put(name, x0, z0, x0+int32(n)-1, z0+int32(n)-1)
}

func (s *session) allowInZone(zone, player string, on bool) error {
	p, found, err := s.gw.playerNamed(player)
	if err != nil {
		return err
	}
	if !found {
		return badZoneArg(fmt.Sprintf("No one is called %q.", player))
	}
	return s.gw.cfg.Zones.Allow(zone, p.ID, on)
}

func (s *session) setZoneMode(zone, mode string) error {
	m := access.Mode(mode)
	if mode == "any" {
		m = access.ModeDefault
	}
	if !m.Valid() {
		return badZoneArg("The mode is creative, survival or any.")
	}
	return s.gw.cfg.Zones.SetMode(zone, m)
}

// listZones tells the admin every zone with its corners, mode and builders.
func (s *session) listZones(ctx context.Context) error {
	all := s.gw.cfg.Zones.All()
	if len(all) == 0 {
		return s.notice(ctx, "There are no zones. "+zoneHelp)
	}
	players, err := s.gw.cfg.Access.Players()
	if err != nil {
		return err
	}
	names := make(map[int64]string, len(players))
	for _, p := range players {
		names[p.ID] = p.Name
	}
	lines := make([]string, 0, len(all))
	for _, z := range all {
		lines = append(lines, describeZone(z, names))
	}
	return s.notice(ctx, strings.Join(lines, "; "))
}

func describeZone(z zones.Zone, names map[int64]string) string {
	out := fmt.Sprintf("%s %d,%d to %d,%d", z.Name, z.X0, z.Z0, z.X1, z.Z1)
	if z.Mode != access.ModeDefault {
		out += " " + string(z.Mode)
	}
	builders := make([]string, 0, len(z.Builders))
	for _, id := range z.Builders {
		builders = append(builders, names[id])
	}
	if len(builders) > 0 {
		out += " for " + strings.Join(builders, ", ")
	}
	return out
}
