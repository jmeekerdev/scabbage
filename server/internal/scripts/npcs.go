package scripts

import (
	"slices"

	"scabbage/internal/world"
)

// NPC is a character standing on top of its script block.
type NPC struct {
	ID   uint32
	At   world.Pos // the script block it stands on
	Name string
}

// NPCFunc hears about each character that appears, is renamed or, when gone is set, leaves.
type NPCFunc func(n NPC, gone bool)

// npcs are the characters by script block, kept in memory so the database is written after scripts run.
type npcs struct {
	byPos   map[world.Pos]NPC
	lastID  uint32
	unsaved map[world.Pos]struct{} // renamed since the last write
	notify  NPCFunc
}

// loadNPCs reads every character saved with its script.
func (s *Service) loadNPCs() error {
	rows, err := s.db.Query(`SELECT x, y, z, npc FROM scripts WHERE npc != ''`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var at world.Pos
		var name string
		if err := rows.Scan(&at.X, &at.Y, &at.Z, &name); err != nil {
			return err
		}
		s.npcs.lastID++
		s.npcs.byPos[at] = NPC{ID: s.npcs.lastID, At: at, Name: name}
	}
	return rows.Err()
}

// NPCs lists every character.
func (s *Service) NPCs() []NPC {
	out := make([]NPC, 0, len(s.npcs.byPos))
	for _, n := range s.npcs.byPos {
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b NPC) int { return int(a.ID) - int(b.ID) })
	return out
}

// NPCByID finds a character by its id.
func (s *Service) NPCByID(id uint32) (NPC, bool) {
	for _, n := range s.npcs.byPos {
		if n.ID == id {
			return n, true
		}
	}
	return NPC{}, false
}

// SetNPC stands a character called name on the script block at `at`, or renames the one there.
func (s *Service) SetNPC(at world.Pos, name string) {
	n, ok := s.npcs.byPos[at]
	if ok && n.Name == name {
		return
	}
	if !ok {
		s.npcs.lastID++
		n = NPC{ID: s.npcs.lastID, At: at}
	}
	n.Name = name
	s.npcs.byPos[at] = n
	s.npcs.unsaved[at] = struct{}{}
	s.tell(n, false)
}

// dropNPC sends away the character at `at`; its row is rewritten or deleted by the caller.
func (s *Service) dropNPC(at world.Pos) {
	n, ok := s.npcs.byPos[at]
	if !ok {
		return
	}
	delete(s.npcs.byPos, at)
	delete(s.npcs.unsaved, at)
	s.tell(n, true)
}

func (s *Service) tell(n NPC, gone bool) {
	if s.npcs.notify != nil {
		s.npcs.notify(n, gone)
	}
}

// saveNPCs writes the names given since the last write.
func (s *Service) saveNPCs() []Failure {
	var out []Failure
	for at := range s.npcs.unsaved {
		delete(s.npcs.unsaved, at)
		n, ok := s.npcs.byPos[at]
		if !ok {
			continue
		}
		if _, err := s.db.Exec(`UPDATE scripts SET npc = ? WHERE x = ? AND y = ? AND z = ?`, n.Name, at.X, at.Y, at.Z); err != nil {
			out = append(out, s.failure(at, err))
		}
	}
	return out
}

// NPCAt is the character standing on the script block at `at`, if there is one.
func (s *Service) NPCAt(at world.Pos) (NPC, bool) {
	n, ok := s.npcs.byPos[at]
	return n, ok
}
