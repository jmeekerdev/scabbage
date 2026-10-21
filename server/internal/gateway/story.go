package gateway

import (
	"errors"
	"time"

	"scabbage/internal/edit"
	"scabbage/internal/journal"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/scripts"
	"scabbage/internal/world"
)

var errTakeSurvival = errors.New("only admins' scripts take items from survival players")

// question is what a script asked a player, waiting for their Answer.
type question struct {
	id      uint32
	at      world.Pos // the asking script block
	choices []string
}

func npcProto(n scripts.NPC) *pb.Npc {
	return &pb.Npc{Id: n.ID, Name: n.Name, Pos: &pb.BlockPos{X: n.At.X, Y: n.At.Y + 1, Z: n.At.Z}}
}

// npcChanged tells everyone a character appeared, was renamed or left; scripts call it with editMu held.
func (g *Gateway) npcChanged(n scripts.NPC, gone bool) {
	out := &pb.NpcsChanged{}
	if gone {
		out.Gone = []uint32{n.ID}
	} else {
		out.Npcs = []*pb.Npc{npcProto(n)}
	}
	g.broadcast(&pb.ServerMessage{Body: &pb.ServerMessage_NpcsChanged{NpcsChanged: out}})
}

// npcsMessage lists every character for a joining player; nil when there are none.
func (g *Gateway) npcsMessage() *pb.ServerMessage {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	all := g.scripts.NPCs()
	if len(all) == 0 {
		return nil
	}
	out := &pb.NpcsChanged{}
	for _, n := range all {
		out.Npcs = append(out.Npcs, npcProto(n))
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_NpcsChanged{NpcsChanged: out}}
}

func questLogMessage(j *journal.Journal) *pb.ServerMessage {
	out := &pb.QuestLog{}
	for _, q := range j.Quests() {
		out.Quests = append(out.Quests, &pb.Quest{Title: q.Title, Step: q.Step, Done: q.Done})
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_QuestLog{QuestLog: out}}
}

// talkTo runs on_talk in the script of a character within reach.
func (s *session) talkTo(m *pb.TalkTo) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no voice")
	}
	now := time.Now()
	if !s.edits.Take(now) {
		return nil
	}
	g := s.gw
	g.editMu.Lock()
	defer g.editMu.Unlock()
	n, ok := g.scripts.NPCByID(m.Npc)
	if ok && edit.InReach(s.body.Pos(), n.At.X, n.At.Y+1, n.At.Z) {
		g.reportScriptsLocked(g.scripts.Talk(n.At, s.player.Name, now), s)
	}
	return nil
}

// answer hands the script that asked the player a question the choice they picked.
func (s *session) answer(m *pb.Answer) error {
	if s.console || !s.edits.Take(time.Now()) {
		return nil
	}
	g := s.gw
	g.mu.Lock()
	a := g.players[s]
	var q *question
	if a != nil && a.asked != nil && a.asked.id == m.Dialogue && int(m.Choice) < len(a.asked.choices) {
		q, a.asked = a.asked, nil
	}
	g.mu.Unlock()
	if q == nil {
		return nil
	}
	g.editMu.Lock()
	defer g.editMu.Unlock()
	g.reportScriptsLocked(g.scripts.Answer(q.at, s.player.Name, q.choices[m.Choice], time.Now()), s)
	return nil
}

// NPC stands a character on the script block.
func (h scriptHost) NPC(name string) error {
	h.g.scripts.SetNPC(h.at, name)
	return nil
}

// Talk shows a nearby player what the script's character says; with choices, it waits for their answer.
func (h scriptHost) Talk(player, text string, choices []string) error {
	speaker := "Script"
	if n, ok := h.g.scripts.NPCAt(h.at); ok {
		speaker = n.Name
	}
	h.g.mu.Lock()
	defer h.g.mu.Unlock()
	s, a := h.nearLocked(player)
	if a == nil {
		return errNotHere
	}
	h.g.dialogue++
	a.asked = nil
	if len(choices) > 0 {
		a.asked = &question{id: h.g.dialogue, at: h.at, choices: choices}
	}
	s.push(&pb.ServerMessage{Body: &pb.ServerMessage_Dialogue{Dialogue: &pb.Dialogue{
		Id: h.g.dialogue, Speaker: speaker, Text: text, Choices: choices,
	}}})
	return nil
}

// Remember keeps something about a nearby player for any script to recall.
func (h scriptHost) Remember(player, key, value string) error {
	return h.withJournal(player, func(_ *session, j *journal.Journal) error { return j.Remember(key, value) })
}

// Recall is what scripts remembered about a nearby player, or "" for nothing.
func (h scriptHost) Recall(player, key string) (value string, err error) {
	err = h.withJournal(player, func(_ *session, j *journal.Journal) error {
		value, _ = j.Recall(key)
		return nil
	})
	return value, err
}

// SetQuest starts or updates a quest in a nearby player's log.
func (h scriptHost) SetQuest(player, title, step string) error {
	return h.changeQuests(player, func(j *journal.Journal) (bool, error) { return j.SetQuest(title, step) })
}

// FinishQuest marks a quest in a nearby player's log done.
func (h scriptHost) FinishQuest(player, title string) error {
	return h.changeQuests(player, func(j *journal.Journal) (bool, error) { return j.Finish(title) })
}

func (h scriptHost) changeQuests(player string, change func(*journal.Journal) (bool, error)) error {
	return h.withJournal(player, func(s *session, j *journal.Journal) error {
		changed, err := change(j)
		if changed {
			s.push(questLogMessage(j))
		}
		return err
	})
}

func (h scriptHost) withJournal(player string, use func(*session, *journal.Journal) error) error {
	h.g.mu.Lock()
	defer h.g.mu.Unlock()
	s, a := h.nearLocked(player)
	if a == nil {
		return errNotHere
	}
	return use(s, a.journal)
}

// TakeItem takes blocks from a nearby player for a quest; creative players have everything, so lose nothing,
// and only admins' scripts take from survival players.
func (h scriptHost) TakeItem(player, item string, count int) (bool, error) {
	b, ok := world.Builtin(item)
	if !ok || b == world.Air {
		return false, errNoBlock
	}
	h.g.mu.Lock()
	defer h.g.mu.Unlock()
	s, a := h.nearLocked(player)
	switch {
	case a == nil:
		return false, errNotHere
	case a.mode == creative:
		return true, nil
	case !h.admin:
		return false, errTakeSurvival
	case !a.inv.Remove(b, count):
		return false, nil
	}
	s.push(inventoryMessage(&a.inv))
	return true, nil
}
