package script

import (
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

const (
	// MaxName is the longest character name.
	MaxName = 24
	// MaxChoices is how many answers ask() may offer.
	MaxChoices = 4
	// MaxChoice is the longest answer ask() may offer.
	MaxChoice = 40
	// MaxAsks is how many players a script may wait on answers from; asking more forgets one.
	MaxAsks = 8
	// MaxKey and MaxValue bound what remember() keeps.
	MaxKey   = 40
	MaxValue = 200
	// MaxTitle and MaxStep bound a quest's title and its next step.
	MaxTitle = 40
	MaxStep  = 120
)

// story is the calls for characters and quests: npc, talk, ask, remember, recall, quest,
// finish_quest and take_item.
func (s *Script) story() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"npc":          s.npc,
		"talk":         s.talk,
		"ask":          s.ask,
		"remember":     s.remember,
		"recall":       s.recall,
		"quest":        s.quest,
		"finish_quest": s.finishQuest,
		"take_item":    s.takeItem,
	}
}

func (s *Script) npc(L *lua.LState) int {
	name := clip(L.CheckString(1), MaxName)
	if name == "" {
		L.ArgError(1, "a character needs a name")
	}
	refuse(L, s.host.NPC(name))
	return 0
}

func (s *Script) talk(L *lua.LState) int {
	refuse(L, s.host.Talk(L.CheckString(1), clip(L.CheckString(2), MaxSay), nil))
	return 0
}

// ask shows a question with choices, and later runs fn with the one the player picks.
func (s *Script) ask(L *lua.LState) int {
	player, text := L.CheckString(1), clip(L.CheckString(2), MaxSay)
	choices := checkChoices(L, 3)
	fn := L.CheckFunction(4)
	refuse(L, s.host.Talk(player, text, choices))
	if s.answers == nil {
		s.answers = map[string]*lua.LFunction{}
	}
	key := strings.ToLower(player)
	if _, waiting := s.answers[key]; !waiting && len(s.answers) >= MaxAsks {
		for k := range s.answers {
			delete(s.answers, k) // the player who asked longest ago has likely walked off; any will do
			break
		}
	}
	s.answers[key] = fn
	return 0
}

func checkChoices(L *lua.LState, n int) []string {
	t := L.CheckTable(n)
	if t.Len() < 1 || t.Len() > MaxChoices {
		L.ArgError(n, "offer between 1 and 4 choices")
	}
	choices := make([]string, 0, t.Len())
	for i := 1; i <= t.Len(); i++ {
		v, ok := t.RawGetInt(i).(lua.LString)
		if !ok {
			L.ArgError(n, "choices are words")
		}
		choices = append(choices, clip(string(v), MaxChoice))
	}
	return choices
}

// remember keeps a string, number or boolean about a player for any script to recall; nil forgets.
func (s *Script) remember(L *lua.LState) int {
	player, key := L.CheckString(1), checkKey(L, 2)
	value := encode(L, 3)
	if len(value) > MaxValue {
		L.ArgError(3, "too long to remember")
	}
	refuse(L, s.host.Remember(player, key, value))
	return 0
}

func (s *Script) recall(L *lua.LState) int {
	value, err := s.host.Recall(L.CheckString(1), checkKey(L, 2))
	refuse(L, err)
	L.Push(decode(value))
	return 1
}

func checkKey(L *lua.LState, n int) string {
	key := strings.TrimSpace(L.CheckString(n))
	if key == "" || len(key) > MaxKey {
		L.ArgError(n, "name it in 1 to 40 letters")
	}
	return key
}

// encode keeps a value's kind in its first letter, so 3 comes back a number and "3" a string.
func encode(L *lua.LState, n int) string {
	switch v := L.Get(n).(type) {
	case *lua.LNilType:
		return ""
	case lua.LBool:
		return "b" + strconv.FormatBool(bool(v))
	case lua.LNumber:
		return "n" + strconv.FormatFloat(float64(v), 'g', -1, 64)
	case lua.LString:
		return "s" + string(v)
	}
	L.ArgError(n, "only words, numbers and true or false can be remembered")
	return ""
}

func decode(value string) lua.LValue {
	if value == "" {
		return lua.LNil
	}
	switch rest := value[1:]; value[0] {
	case 'b':
		return lua.LBool(rest == "true")
	case 'n':
		if f, err := strconv.ParseFloat(rest, 64); err == nil {
			return lua.LNumber(f)
		}
	case 's':
		return lua.LString(rest)
	}
	return lua.LNil
}

func (s *Script) quest(L *lua.LState) int {
	player, title := L.CheckString(1), checkTitle(L, 2)
	refuse(L, s.host.SetQuest(player, title, clip(L.CheckString(3), MaxStep)))
	return 0
}

func (s *Script) finishQuest(L *lua.LState) int {
	refuse(L, s.host.FinishQuest(L.CheckString(1), checkTitle(L, 2)))
	return 0
}

func checkTitle(L *lua.LState, n int) string {
	title := clip(L.CheckString(n), MaxTitle)
	if title == "" {
		L.ArgError(n, "a quest needs a title")
	}
	return title
}

func (s *Script) takeItem(L *lua.LState) int {
	player, item := L.CheckString(1), L.CheckString(2)
	count := L.OptInt(3, 1)
	if count < 1 || count > MaxGive {
		L.ArgError(3, "take between 1 and 64")
	}
	took, err := s.host.TakeItem(player, item, count)
	refuse(L, err)
	L.Push(lua.LBool(took))
	return 1
}
