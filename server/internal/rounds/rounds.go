// Package rounds plays an arena's minigame: a lobby that counts down, then a round that ends in a win.
package rounds

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"

	"scabbage/internal/arenas"
)

const (
	Countdown  = 10 * time.Second // from enough players joining to the start
	MaxPlayers = 16
)

// Effect is something the gateway does for one player: tell them something, move them, or both.
type Effect struct {
	Player uint32
	Say    string
	Move   *arenas.Mark // a spawn, or the portal they joined from
}

// Event is a round starting, with who plays, or ending, with who won; scripts in the arena hear them.
type Event struct {
	Start bool
	Names []string // the players at the start, the winners at the end
}

type phase uint8

const (
	waiting phase = iota
	starting
	playing
)

type entry struct {
	id     uint32
	name   string
	lobby  arenas.Mark // the portal they joined from
	spawn  arenas.Mark
	team   arenas.Team
	points int
	out    bool
	placed bool        // stood inside the arena since last sent to spawn
	flag   arenas.Team // whose flag they carry
}

// Round is one arena's lobby and game; it is not safe for concurrent use.
type Round struct {
	arena   arenas.Arena
	phase   phase
	ends    time.Time // when the countdown or game ends; zero for never
	players []*entry
	events  []Event // not yet taken
}

func New(a arenas.Arena) *Round { return &Round{arena: a} }

func (r *Round) Zone() string { return r.arena.Zone }

func (r *Round) Has(id uint32) bool { return r.find(id) != nil }

func (r *Round) Empty() bool { return len(r.players) == 0 }

// Tagging is whether a hit puts a player out, as it does while tag is on.
func (r *Round) Tagging() bool { return r.phase == playing && r.arena.Game == arenas.Tag }

// TakeEvents returns the starts and ends since the last call.
func (r *Round) TakeEvents() []Event {
	out := r.events
	r.events = nil
	return out
}

// AddPoints gives a playing player n points, as a script might; it reports false when they aren't playing.
func (r *Round) AddPoints(name string, n int) ([]Effect, bool) {
	var e *entry
	for _, o := range r.players {
		if strings.EqualFold(o.name, name) && !o.out {
			e = o
		}
	}
	if r.phase != playing || e == nil {
		return nil, false
	}
	e.points = max(0, e.points+n)
	if r.arena.Win != arenas.WinPoints {
		return nil, true
	}
	return r.checkPoints(e), true
}

func (r *Round) find(id uint32) *entry {
	for _, e := range r.players {
		if e.id == id {
			return e
		}
	}
	return nil
}

// need is how many players a round takes: one per team, two to fight, else one.
func (r *Round) need() int {
	if teams := len(r.arena.Teams()); teams > 0 {
		return teams
	}
	if r.arena.Win == arenas.WinLast {
		return 2
	}
	return 1
}

// Join adds a player to the lobby, starting the countdown once there are enough.
func (r *Round) Join(id uint32, name string, lobby arenas.Mark, now time.Time) []Effect {
	switch err := r.arena.Ready(); {
	case r.Has(id):
		return nil
	case err != nil:
		return []Effect{{Player: id, Say: fmt.Sprintf("%s isn't ready: %s.", r.arena.Zone, err)}}
	case r.phase == playing:
		return []Effect{{Player: id, Say: fmt.Sprintf("A round is on in %s. Try again when it ends.", r.arena.Zone)}}
	case len(r.players) >= MaxPlayers:
		return []Effect{{Player: id, Say: r.arena.Zone + " is full."}}
	}
	r.players = append(r.players, &entry{id: id, name: name, lobby: lobby})
	effects := r.tellAll(fmt.Sprintf("%s joined %s.", name, r.arena.Zone))
	if r.phase == waiting && len(r.players) >= r.need() {
		r.phase, r.ends = starting, now.Add(Countdown)
		effects = append(effects, r.tellAll(fmt.Sprintf("%s starts in %d seconds.", r.arena.Zone, int(Countdown.Seconds())))...)
	}
	return effects
}

// Leave takes a player out of the round and back to their portal.
func (r *Round) Leave(id uint32) []Effect {
	e := r.find(id)
	if e == nil {
		return nil
	}
	r.players = slices.DeleteFunc(r.players, func(o *entry) bool { return o == e })
	effects := []Effect{{Player: id, Say: "You left " + r.arena.Zone + ".", Move: &e.lobby}}
	switch {
	case r.Empty():
		r.reset()
	case r.phase == starting && len(r.players) < r.need():
		r.phase, r.ends = waiting, time.Time{}
	case r.phase == playing:
		effects = append(effects, r.checkLast()...)
	}
	return effects
}

// Step starts the game when the countdown ends and finishes it when time runs out.
func (r *Round) Step(now time.Time) []Effect {
	if r.ends.IsZero() || now.Before(r.ends) {
		return nil
	}
	if r.phase == starting {
		return r.start(now)
	}
	return r.timeUp()
}

func (r *Round) start(now time.Time) []Effect {
	r.phase, r.ends = playing, time.Time{}
	if r.arena.Seconds > 0 {
		r.ends = now.Add(time.Duration(r.arena.Seconds) * time.Second)
	}
	teams := r.arena.Teams()
	var effects []Effect
	for i, e := range r.players {
		if len(teams) > 0 {
			e.team = teams[i%len(teams)]
		}
		e.spawn = r.spawnFor(e.team, i)
		effects = append(effects, r.toSpawn(e))
	}
	names := make([]string, len(r.players))
	for i, e := range r.players {
		names[i] = e.name
	}
	r.events = append(r.events, Event{Start: true, Names: names})
	return append(effects, r.tellAll("Go! "+r.aim())...)
}

// spawnFor picks the i-th player's spawn among their team's, or the shared ones.
func (r *Round) spawnFor(team arenas.Team, i int) arenas.Mark {
	var mine []arenas.Mark
	for _, m := range r.arena.Spawns {
		if m.Team == team {
			mine = append(mine, m)
		}
	}
	if len(mine) == 0 {
		mine = r.arena.Spawns
	}
	return mine[i%len(mine)]
}

func (r *Round) toSpawn(e *entry) Effect {
	e.placed = false
	return Effect{Player: e.id, Move: &e.spawn}
}

// aim says how to win.
func (r *Round) aim() string {
	switch r.arena.Game {
	case arenas.Flags:
		return "Bring another team's flag to your goal!"
	case arenas.Tag:
		return "Tag others and don't get tagged!"
	case arenas.Spleef:
		return "Stay on the floor!"
	}
	switch r.arena.Win {
	case arenas.WinGoal:
		return "First to the goal wins!"
	case arenas.WinPoints:
		return fmt.Sprintf("First to %d points wins!", r.arena.Points)
	}
	return "Be the last one standing!"
}

// Stand follows a player to the block their feet are in, scoring goals and catching anyone who leaves.
func (r *Round) Stand(id uint32, x, y, z int32, inside bool) []Effect {
	e := r.find(id)
	if r.phase != playing || e == nil || e.out {
		return nil
	}
	if !inside {
		if !e.placed {
			return nil // still on their way in
		}
		effects, _ := r.Fall(id)
		return effects
	}
	e.placed = true
	for _, g := range r.arena.Goals {
		if g.At(x, y, z) {
			return r.reach(e, g.Team)
		}
	}
	return nil
}

// reach handles a player standing on a goal of a team, or everyone's.
func (r *Round) reach(e *entry, goal arenas.Team) []Effect {
	ours := goal == arenas.NoTeam || goal == e.team
	switch {
	case r.arena.Game == arenas.Flags && !ours && e.flag == arenas.NoTeam:
		e.flag = goal
		return r.tellAll(fmt.Sprintf("%s has the %s flag!", e.name, goal))
	case r.arena.Game == arenas.Flags && goal == e.team && e.flag != arenas.NoTeam:
		e.flag = arenas.NoTeam
		e.points++
		return append(r.tellAll(fmt.Sprintf("%s scored for %s!", e.name, e.team)), r.checkPoints(e)...)
	case r.arena.Game == arenas.Flags || !ours:
		return nil
	case r.arena.Win == arenas.WinGoal:
		return r.finish("", []string{r.side(e)})
	case r.arena.Win == arenas.WinPoints:
		e.points++
		return append([]Effect{r.toSpawn(e)}, r.checkPoints(e)...)
	}
	return nil
}

// Fall is a player fainting, being tagged or leaving the arena: out in fights, else back to spawn.
// It reports false when the round has nothing to do with it, so they respawn as usual.
func (r *Round) Fall(id uint32) ([]Effect, bool) {
	e := r.find(id)
	if r.phase != playing || e == nil || e.out {
		return nil, false
	}
	e.flag = arenas.NoTeam
	if r.arena.Win != arenas.WinLast {
		return []Effect{r.toSpawn(e)}, true
	}
	effects := r.tellAll(e.name + " is out!")
	e.out = true
	effects = append(effects, Effect{Player: e.id, Move: &e.lobby})
	return append(effects, r.checkLast()...), true
}

// side is who a player wins for: their team, or themselves.
func (r *Round) side(e *entry) string {
	if e.team != arenas.NoTeam {
		return string(e.team)
	}
	return e.name
}

// totals is each side's points, and whether they are still in.
func (r *Round) totals() (map[string]int, map[string]bool) {
	points, in := map[string]int{}, map[string]bool{}
	for _, e := range r.players {
		points[r.side(e)] += e.points
		in[r.side(e)] = in[r.side(e)] || !e.out
	}
	return points, in
}

func (r *Round) checkPoints(e *entry) []Effect {
	if points, _ := r.totals(); points[r.side(e)] >= r.arena.Points {
		return r.finish("", []string{r.side(e)})
	}
	return nil
}

// checkLast ends a fight once at most one side is still in.
func (r *Round) checkLast() []Effect {
	if r.arena.Win != arenas.WinLast {
		return nil
	}
	if left := r.sidesIn(); len(left) <= 1 {
		return r.finish("", left)
	}
	return nil
}

// sidesIn lists the sides still in, in join order.
func (r *Round) sidesIn() []string {
	var out []string
	for _, e := range r.players {
		if !e.out && !slices.Contains(out, r.side(e)) {
			out = append(out, r.side(e))
		}
	}
	return out
}

// timeUp ends the round: the most points win, or whoever is still in a fight; no one wins a race.
func (r *Round) timeUp() []Effect {
	switch r.arena.Win {
	case arenas.WinLast:
		return r.finish("Time's up! ", r.sidesIn())
	case arenas.WinPoints:
		return r.finish("Time's up! ", r.leaders())
	}
	return r.finish("Time's up! ", nil)
}

// leaders are the sides with the most points, if anyone scored.
func (r *Round) leaders() []string {
	points, _ := r.totals()
	best := 0
	for _, p := range points {
		best = max(best, p)
	}
	var out []string
	for _, e := range r.players {
		if s := r.side(e); best > 0 && points[s] == best && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// finish announces the winners, sends everyone still in back to their portal and empties the round.
func (r *Round) finish(why string, winners []string) []Effect {
	r.events = append(r.events, Event{Names: winners})
	effects := r.tellAll(why + result(winners))
	for _, e := range r.players {
		if !e.out {
			effects = append(effects, Effect{Player: e.id, Move: &e.lobby})
		}
	}
	r.reset()
	return effects
}

// result is "Red team wins!", "It's a tie between Ann and Bo." or "No one won."
func result(winners []string) string {
	names := make([]string, len(winners))
	for i, w := range winners {
		names[i] = w
		if arenas.Team(w).Valid() {
			names[i] = capitalize(w) + " team"
		}
	}
	switch len(names) {
	case 0:
		return "No one won."
	case 1:
		return names[0] + " wins!"
	}
	return "It's a tie between " + strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1] + "."
}

func (r *Round) reset() { r.phase, r.ends, r.players = waiting, time.Time{}, nil }

// tellAll says text to everyone in the round.
func (r *Round) tellAll(text string) []Effect {
	out := make([]Effect, len(r.players))
	for i, e := range r.players {
		out[i] = Effect{Player: e.id, Say: text}
	}
	return out
}

// Board is what the round's players see of it.
type Board struct {
	Zone    string
	Status  string
	Seconds int // left on the clock; 0 for none
	Scores  []Score
}

type Score struct {
	Name   string
	Team   arenas.Team
	Points int
	Out    bool
	Flag   arenas.Team
}

func (r *Round) Board(now time.Time) Board {
	b := Board{Zone: r.arena.Zone, Status: "Playing"}
	switch r.phase {
	case waiting:
		b.Status = fmt.Sprintf("Waiting for %d more", r.need()-len(r.players))
	case starting:
		b.Status = "Starting"
	}
	if !r.ends.IsZero() {
		b.Seconds = int(math.Ceil(r.ends.Sub(now).Seconds()))
	}
	for _, e := range r.players {
		b.Scores = append(b.Scores, Score{Name: e.name, Team: e.team, Points: e.points, Out: e.out, Flag: e.flag})
	}
	return b
}

func capitalize(s string) string {
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
