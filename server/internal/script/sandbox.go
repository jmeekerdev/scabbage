package script

import (
	"regexp"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// unsafeGlobals reach files, load code from strings or poke at the VM; scripts never get them.
var unsafeGlobals = []string{
	"collectgarbage", "dofile", "getfenv", "setfenv", "load", "loadfile", "loadstring",
	"print", "_printregs", "module", "require", "newproxy",
}

// maxFormatWidth bounds string.format widths, so "%999999999d" can't fill memory in one step.
const maxFormatWidth = 99

var formatWidths = regexp.MustCompile(`%[-+ #0]*(\d*)(?:\.(\d*))?`)

// newSandbox is a Lua state with only base, string, table and math, and small fixed stacks.
func newSandbox() *lua.LState {
	L := lua.NewState(lua.Options{
		SkipOpenLibs:        true,
		CallStackSize:       64,
		RegistrySize:        1024,
		RegistryMaxSize:     16 * 1024,
		MinimizeStackMemory: true,
	})
	for _, lib := range []struct {
		name string
		open lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.StringLibName, lua.OpenString},
		{lua.TabLibName, lua.OpenTable},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.open))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	for _, name := range unsafeGlobals {
		L.SetGlobal(name, lua.LNil)
	}
	limitStrings(L)
	return L
}

// limitStrings drops string.rep and caps string.format widths, the two ways one call makes a huge string.
func limitStrings(L *lua.LState) {
	strs := L.GetGlobal(lua.StringLibName).(*lua.LTable)
	strs.RawSetString("rep", lua.LNil)
	format := strs.RawGetString("format").(*lua.LFunction).GFunction
	strs.RawSetString("format", L.NewFunction(func(L *lua.LState) int {
		for _, m := range formatWidths.FindAllStringSubmatch(L.CheckString(1), -1) {
			for _, n := range m[1:] {
				if w, _ := strconv.Atoi(n); w > maxFormatWidth {
					L.RaiseError("string.format widths go up to %d", maxFormatWidth)
				}
			}
		}
		return format(L)
	}))
}
