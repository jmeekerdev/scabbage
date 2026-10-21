package script

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"

	lua "github.com/yuin/gopher-lua"
)

// Host is the world as scripts touch it; it checks zones and reach and says no with an error.
type Host interface {
	Say(text string)
	SetBlock(x, y, z int32, name string) error
	GiveItem(player, item string, count int) error
	Teleport(player string, x, y, z float64) error
	AddPoints(player string, n int) error // in the round of the arena the script is in
	Where() (x, y, z int32)               // the script's own block
	NPC(name string) error                // stands a character called name on the script block
	Talk(player, text string, choices []string) error
	Remember(player, key, value string) error // an empty value forgets
	Recall(player, key string) (string, error)
	SetQuest(player, title, step string) error
	FinishQuest(player, title string) error
	TakeItem(player, item string, count int) (bool, error) // false when they don't have that many
}

const (
	// MaxSay is the longest message a script may say.
	MaxSay = 200
	// MaxGive is the most of an item one give_item call hands over.
	MaxGive = 64
	// MaxPoints is the most points one add_points call gives or takes.
	MaxPoints = 10
	// minDelay and maxDelay bound how far ahead after() may schedule.
	minDelay = 100 * time.Millisecond
	maxDelay = time.Hour
)

// register gives the script its world calls (say, set_block, give_item, teleport, add_points, after
// and the story calls) and `here`, where its block is.
func (s *Script) register() {
	x, y, z := s.host.Where()
	here := s.L.NewTable()
	here.RawSetString("x", lua.LNumber(x))
	here.RawSetString("y", lua.LNumber(y))
	here.RawSetString("z", lua.LNumber(z))
	s.L.SetGlobal("here", here)
	for name, fn := range map[string]lua.LGFunction{
		"say":        s.say,
		"set_block":  s.setBlock,
		"give_item":  s.giveItem,
		"teleport":   s.teleport,
		"add_points": s.addPoints,
		"after":      s.after,
	} {
		s.L.SetGlobal(name, s.L.NewFunction(s.counted(fn)))
	}
	for name, fn := range s.story() {
		s.L.SetGlobal(name, s.L.NewFunction(s.counted(fn)))
	}
}

// counted makes fn use up one of the run's actions, failing the run once they are gone.
func (s *Script) counted(fn lua.LGFunction) lua.LGFunction {
	return func(L *lua.LState) int {
		if s.actions >= MaxActions {
			L.RaiseError("a script may do %d things at a time", MaxActions)
		}
		s.actions++
		return fn(L)
	}
}

func (s *Script) say(L *lua.LState) int {
	s.host.Say(clip(L.CheckString(1), MaxSay))
	return 0
}

// clip trims text and cuts it to at most n characters.
func clip(text string, n int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > n {
		text = string([]rune(text)[:n])
	}
	return text
}

func (s *Script) setBlock(L *lua.LState) int {
	x, y, z := checkInt32(L, 1), checkInt32(L, 2), checkInt32(L, 3)
	refuse(L, s.host.SetBlock(x, y, z, L.CheckString(4)))
	return 0
}

func (s *Script) giveItem(L *lua.LState) int {
	player, item := L.CheckString(1), L.CheckString(2)
	count := L.OptInt(3, 1)
	if count < 1 || count > MaxGive {
		L.ArgError(3, "give between 1 and 64")
	}
	refuse(L, s.host.GiveItem(player, item, count))
	return 0
}

func (s *Script) teleport(L *lua.LState) int {
	player := L.CheckString(1)
	x, y, z := checkFinite(L, 2), checkFinite(L, 3), checkFinite(L, 4)
	refuse(L, s.host.Teleport(player, x, y, z))
	return 0
}

func (s *Script) addPoints(L *lua.LState) int {
	player, n := L.CheckString(1), L.OptInt(2, 1)
	if n < -MaxPoints || n > MaxPoints || n == 0 {
		L.ArgError(2, "give between -10 and 10 points")
	}
	refuse(L, s.host.AddPoints(player, n))
	return 0
}

// after runs fn once, the given number of seconds from now.
func (s *Script) after(L *lua.LState) int {
	delay := time.Duration(checkFinite(L, 1) * float64(time.Second))
	fn := L.CheckFunction(2)
	if delay < minDelay || delay > maxDelay {
		L.ArgError(1, "wait between 0.1 seconds and an hour")
	}
	if len(s.timers) >= MaxTimers {
		L.RaiseError("a script may wait on %d timers at once", MaxTimers)
	}
	s.timers = append(s.timers, timer{at: s.now.Add(delay), fn: fn})
	return 0
}

func checkInt32(L *lua.LState, n int) int32 {
	v := checkFinite(L, n)
	if v < math.MinInt32 || v > math.MaxInt32 {
		L.ArgError(n, "too far away")
	}
	return int32(math.Floor(v))
}

func checkFinite(L *lua.LState, n int) float64 {
	v := float64(L.CheckNumber(n))
	if math.IsNaN(v) || math.IsInf(v, 0) {
		L.ArgError(n, "not a real number")
	}
	return v
}

// refuse turns the host saying no into a Lua error the script can catch with pcall.
func refuse(L *lua.LState, err error) {
	if err != nil {
		L.RaiseError("%s", err.Error())
	}
}
