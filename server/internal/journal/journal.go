// Package journal keeps what scripts remember about each player and the player's quest log.
package journal

import (
	"database/sql"
	"errors"
	"maps"
	"slices"
)

const (
	// MaxMemories is how many things scripts may remember about one player.
	MaxMemories = 100
	// MaxQuests is how many quests one player's log holds.
	MaxQuests = 50
)

// ErrFull means the player's memories or quest log have no room left.
var ErrFull = errors.New("that player's journal is full")

// Quest is one entry in a quest log.
type Quest struct {
	Title string
	Step  string // what to do next; empty once done
	Done  bool
}

// Journal is one player's memories and quests; it is not safe for concurrent use.
type Journal struct {
	memories map[string]string
	quests   []Quest // newest first
	dirty    bool
}

// Saved is a copy of a journal to write to the database.
type Saved struct {
	Memories map[string]string
	Quests   []Quest
}

// Remember keeps value under key; an empty value forgets it.
func (j *Journal) Remember(key, value string) error {
	old, had := j.memories[key]
	switch {
	case value == "":
		if had {
			delete(j.memories, key)
			j.dirty = true
		}
		return nil
	case old == value:
		return nil
	case !had && len(j.memories) >= MaxMemories:
		return ErrFull
	}
	if j.memories == nil {
		j.memories = map[string]string{}
	}
	j.memories[key] = value
	j.dirty = true
	return nil
}

// Recall is the value remembered under key.
func (j *Journal) Recall(key string) (string, bool) {
	v, ok := j.memories[key]
	return v, ok
}

// SetQuest starts or updates a quest and moves it to the top, reporting whether anything changed.
func (j *Journal) SetQuest(title, step string) (bool, error) {
	return j.put(Quest{Title: title, Step: step})
}

// Finish marks a quest done, adding it if it was never started.
func (j *Journal) Finish(title string) (bool, error) {
	return j.put(Quest{Title: title, Done: true})
}

// Quests is the log, newest first.
func (j *Journal) Quests() []Quest { return slices.Clone(j.quests) }

func (j *Journal) put(q Quest) (bool, error) {
	i := slices.IndexFunc(j.quests, func(o Quest) bool { return o.Title == q.Title })
	if i >= 0 && j.quests[i] == q {
		return false, nil
	}
	if i >= 0 {
		j.quests = slices.Delete(j.quests, i, i+1)
	} else if len(j.quests) >= MaxQuests && !j.dropOldestDone() {
		return false, ErrFull
	}
	j.quests = slices.Insert(j.quests, 0, q)
	j.dirty = true
	return true, nil
}

func (j *Journal) dropOldestDone() bool {
	for i := len(j.quests) - 1; i >= 0; i-- {
		if j.quests[i].Done {
			j.quests = slices.Delete(j.quests, i, i+1)
			return true
		}
	}
	return false
}

// Unsaved hands out a copy to save if the journal changed since the last one.
func (j *Journal) Unsaved() (Saved, bool) {
	if !j.dirty {
		return Saved{}, false
	}
	j.dirty = false
	return Saved{Memories: maps.Clone(j.memories), Quests: j.Quests()}, true
}

// Load reads a player's journal.
func Load(db *sql.DB, player int64) (*Journal, error) {
	j := &Journal{memories: map[string]string{}}
	if err := eachRow(db, `SELECT key, value FROM memories WHERE player_id = ?`, player, func(r *sql.Rows) error {
		var k, v string
		err := r.Scan(&k, &v)
		j.memories[k] = v
		return err
	}); err != nil {
		return nil, err
	}
	err := eachRow(db, `SELECT title, step, done FROM quests WHERE player_id = ? ORDER BY place`, player, func(r *sql.Rows) error {
		var q Quest
		err := r.Scan(&q.Title, &q.Step, &q.Done)
		j.quests = append(j.quests, q)
		return err
	})
	return j, err
}

func eachRow(db *sql.DB, query string, player int64, scan func(*sql.Rows) error) error {
	rows, err := db.Query(query, player)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Store replaces a player's saved journal with s.
func Store(db *sql.DB, player int64, s Saved) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, q := range []string{`DELETE FROM memories WHERE player_id = ?`, `DELETE FROM quests WHERE player_id = ?`} {
		if _, err := tx.Exec(q, player); err != nil {
			return err
		}
	}
	for k, v := range s.Memories {
		if _, err := tx.Exec(`INSERT INTO memories (player_id, key, value) VALUES (?, ?, ?)`, player, k, v); err != nil {
			return err
		}
	}
	for i, q := range s.Quests {
		if _, err := tx.Exec(`INSERT INTO quests (player_id, title, step, done, place) VALUES (?, ?, ?, ?, ?)`,
			player, q.Title, q.Step, q.Done, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}
