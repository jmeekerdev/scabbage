// Package zones keeps protected areas where only admins and chosen builders may change blocks.
package zones

import (
	"database/sql"
	"errors"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"scabbage/internal/access"
)

// SpawnName is the zone around the world spawn; it exists unless the server turns it off.
const SpawnName = "Spawn"

const maxNameRunes = 24

var (
	ErrNoZone  = errors.New("no such zone")
	ErrBadName = errors.New("zone names are 1 to 24 characters")
)

// Zone is a column of the world from (X0, Z0) to (X1, Z1), corners included.
type Zone struct {
	Name           string
	X0, Z0, X1, Z1 int32
	Mode           access.Mode // the game mode inside; default keeps each player's own
	Builders       []int64     // players besides admins who may build here
}

func (z Zone) Contains(x, zz int32) bool { return x >= z.X0 && x <= z.X1 && zz >= z.Z0 && zz <= z.Z1 }

func (z Zone) Lets(player int64) bool { return slices.Contains(z.Builders, player) }

func (z Zone) area() int64 { return int64(z.X1-z.X0+1) * int64(z.Z1-z.Z0+1) }

// Service holds every zone in memory and saves changes to the database.
type Service struct {
	db    *sql.DB
	spawn *Zone // the built-in spawn zone, used until one is saved under its name
	mu    sync.RWMutex
	zones []Zone
}

// New loads the saved zones; spawn is the built-in spawn zone's half-width, 0 for none.
func New(db *sql.DB, spawn int32) (*Service, error) {
	s := &Service{db: db}
	if spawn > 0 {
		s.spawn = &Zone{Name: SpawnName, X0: -spawn, Z0: -spawn, X1: spawn, Z1: spawn}
	}
	return s, s.load()
}

// All lists every zone, the built-in spawn zone included.
func (s *Service) All() []Zone {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := slices.Clone(s.zones)
	if s.spawn != nil && !s.saved(SpawnName) {
		out = append(out, *s.spawn)
	}
	return out
}

// At finds the zone covering a column; where zones overlap the smallest wins.
func (s *Service) At(x, z int32) (Zone, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best Zone
	found := false
	pick := func(zone Zone) {
		if zone.Contains(x, z) && (!found || zone.area() < best.area()) {
			best, found = zone, true
		}
	}
	for _, zone := range s.zones {
		pick(zone)
	}
	if s.spawn != nil && !s.saved(SpawnName) {
		pick(*s.spawn)
	}
	return best, found
}

// CanBuild reports whether a player may change blocks in the column at x, z.
func (s *Service) CanBuild(player int64, admin bool, x, z int32) bool {
	zone, ok := s.At(x, z)
	return admin || !ok || zone.Lets(player)
}

// Put makes a zone between two corners, or moves the one with that name there.
func (s *Service) Put(name string, x0, z0, x1, z1 int32) error {
	if n := utf8.RuneCountInString(name); n == 0 || n > maxNameRunes {
		return ErrBadName
	}
	z, ok := s.Named(name)
	if !ok {
		z = Zone{Name: name}
	}
	z.X0, z.X1 = min(x0, x1), max(x0, x1)
	z.Z0, z.Z1 = min(z0, z1), max(z0, z1)
	return s.save(z)
}

// SetMode sets the game mode players have inside a zone.
func (s *Service) SetMode(name string, m access.Mode) error {
	if !m.Valid() {
		return access.ErrBadMode
	}
	z, ok := s.Named(name)
	if !ok {
		return ErrNoZone
	}
	z.Mode = m
	return s.save(z)
}

// Allow adds a builder to a zone, or takes them off when on is false.
func (s *Service) Allow(name string, player int64, on bool) error {
	z, ok := s.Named(name)
	if !ok {
		return ErrNoZone
	}
	z.Builders = slices.DeleteFunc(slices.Clone(z.Builders), func(p int64) bool { return p == player })
	if on {
		z.Builders = append(z.Builders, player)
	}
	return s.save(z)
}

// Remove deletes a saved zone; removing a changed spawn zone brings back the built-in one.
func (s *Service) Remove(name string) error {
	res, err := s.db.Exec(`DELETE FROM zones WHERE name = ?`, name)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNoZone
	}
	return s.load()
}

// Named finds a zone by name, ignoring case.
func (s *Service) Named(name string) (Zone, bool) {
	for _, z := range s.All() {
		if strings.EqualFold(z.Name, name) {
			return z, true
		}
	}
	return Zone{}, false
}

// saved reports whether a zone by this name is stored; call with mu held.
func (s *Service) saved(name string) bool {
	return slices.ContainsFunc(s.zones, func(z Zone) bool { return strings.EqualFold(z.Name, name) })
}
