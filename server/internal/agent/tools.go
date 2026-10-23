package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	// MaxBoxBlocks caps the boxes look, clear and fill take: a 32-block cube.
	MaxBoxBlocks = 32 * 32 * 32
	// MaxPlace caps the cells one place call sets.
	MaxPlace = 4096
)

// ways explains MCP and plain HTTP, and when each suits, for the server at base.
const ways = `Two ways in, with the same tools: MCP at %[1]s/mcp, and plain HTTP: POST
%[1]s/api/agent/v1/{tool} with a JSON object of arguments and the header
Authorization: Bearer <token> (GET %[1]s/api/agent/v1/ explains it).
Use tool calls to look around, check status and make small edits. For big designs, hundreds or
thousands of cells, write a short script that computes the cells and POSTs them to
%[1]s/api/agent/v1/place in batches of up to 4,096, instead of writing every cell into a tool call.
Over MCP you may not have the token: ask the player, who gets one in game with /agent link.`

// InstructionsFor is Instructions plus how to reach the server at base, e.g. "https://host".
func InstructionsFor(base string) string {
	return Instructions + "\n\n" + fmt.Sprintf(ways, base)
}

// baseURL is the scheme and host the agent reached this server at, behind a proxy too.
func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// Tool is one thing an agent can ask for, with a JSON Schema for its arguments.
type Tool struct {
	Name        string
	Description string
	Input       map[string]any
}

// Caller runs a tool for a token's owner. A Refusal is shown to the agent; other errors are not.
type Caller interface {
	Call(ctx context.Context, o Owner, tool string, args json.RawMessage) (any, error)
}

// Refusal is an error the agent should read and act on, like "open Scabbage in a browser first".
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Instructions tell an agent how the tools fit together.
const Instructions = `You drive the player's own avatar in Scabbage, a voxel game, while they watch.
Coordinates are whole blocks, 3 to a meter; y is up and north is -z. The avatar is about 5 blocks
tall and 2 wide, so a doorway is 2 wide and 6 tall.
Start with whereami (where you are) and blocks (every block name, with its shape and whether it
is see-through: glass for windows, stairs turned each way, slabs, lamps). "air" is a block too:
fill a box with air to cut a doorway or window, or give cells "air" in place.
Use look to see the ground before building.
Like the player, the avatar edits with a brush: a cube 3, 2 or 1 blocks on a side. Jobs pick the
biggest brushes that fit by themselves, about 8 brushes a second, so one fill of a big box is far
quicker than many small calls, and boxes whose sides are multiples of 3 go fastest (a 9×9×3 wall
is 9 brushes). Job tools answer with the brushes they need and the seconds that takes on the spot.
For a cabin, tower, wall, floor or stairs, build makes one from a template in a single call.
goto, clear, fill, place, build and undo start a job and return at once; then call status with
wait: true until its state is done or stopped (each wait lasts up to 20 seconds). One job runs at a
time, and the player stops it by moving.
Jobs that change blocks usually wait for the player first: they see a see-through preview and say
/yes or /no in game. Then the job's answer has awaiting: true and status says
awaiting_confirmation; tell the player in chat what you're about to build so they know to answer. The avatar flies to the work itself, over the trees when it finds no way
through, so there's no need to goto first. Edits follow the player's own rules.
Every job really changes the world wherever its cells are, even high in the sky: there's no dry
run, so build only where you mean to. undo takes back the last job.
The player can talk to you from the game: /ai MESSAGE reaches only you, through listen. When asked
to help in game, loop: listen, do what the message asks (asking back with say private: true when
unsure), report with say, then listen again.
To design a new building, write it as a plan and propose it: the player sees it as a ghost and
says /yes to build it or /no, maybe with a reason that status shows in its note. The player may
also just answer in chat, like "make the roof steeper": nothing reaches you unless you ask, so
after proposing or asking a question, keep calling chat with wait: true and after: the last id you
saw (or status with wait: true) until they answer. Propose again to show a revised plan in its place.
When they like it, save_plan keeps it, and anyone can then build it by name: you with build, the
player with /build NAME. They can share it with everyone from the game.
A plan is a palette of one-character keys naming blocks, and layers from the bottom up, each a list
of rows from the front to the back, each row a string of keys across the front from left to right
(as seen facing the front). "." leaves a cell alone and "_" makes it air.
Every layer has the same rows and every row the same length, at most 32 each way.
Placed at x,y,z facing a way, the plan's front row's middle bottom cell lands at x,y,z, and its
front faces that way. material swaps the block the plan uses most.`

type schema = map[string]any

var (
	integer = schema{"type": "integer"}
	point   = object(schema{"x": integer, "y": integer, "z": integer}, "x", "y", "z")
	box     = schema{"from": point, "to": point}
	noArgs  = object(schema{})
	facing  = schema{"type": "string", "enum": []string{"north", "east", "south", "west"}}

	planName  = schema{"type": "string", "pattern": "^[a-z][a-z0-9_-]{1,23}$"}
	planInput = schema{"oneOf": []any{
		schema{"type": "string", "description": "a saved plan's name"},
		object(schema{
			"name":    planName,
			"about":   schema{"type": "string", "maxLength": 200},
			"palette": schema{"type": "object", "additionalProperties": schema{"type": "string"}},
			"layers": schema{"type": "array", "minItems": 1, "maxItems": 32, "items": schema{
				"type": "array", "minItems": 1, "maxItems": 32, "items": schema{"type": "string", "minLength": 1, "maxLength": 32}}},
		}, "name", "palette", "layers"),
	}}
)

func object(props schema, required ...string) schema {
	s := schema{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func with(props schema, more schema) schema {
	out := schema{}
	for _, m := range []schema{props, more} {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// Tools lists every tool, in the order agents see them.
var Tools = []Tool{
	{"whereami", "Where your avatar is (its feet), which way it faces, whether it flies, its game mode and zone.", noArgs},
	{"blocks", "Every block you can place with fill and place: its name, whether it's solid or see-through " +
		"(glass), glows (lamps), its shape if not a cube (slab, stairs_east, torch, …) and whether it's usable (switches).", noArgs},
	{"look", "Counts the blocks in a box and maps its surface: for each column from x1 to x2 (rows from z1 to z2), " +
		"the height of the top non-air block and its name, or null. At most 32,768 blocks.",
		object(box, "from", "to")},
	{"players", "Who is online, where, in which game mode, and whether they are away (their game is paused).", noArgs},
	{"status", "The current or last job: its state (running, done or stopped), percent done and a note. " +
		"wait: true waits up to 20 seconds for a running job to end.",
		object(schema{"wait": schema{"type": "boolean"}})},
	{"say", "Says something in chat as the player, marked 🤖 so everyone knows an agent said it. " +
		"private: true says it only to the player, e.g. to answer their /ai message.",
		object(schema{"text": schema{"type": "string", "minLength": 1, "maxLength": 200},
			"private": schema{"type": "boolean"}}, "text")},
	{"listen", "The player's messages for you, typed in game as /ai MESSAGE, each handed over once, oldest first. " +
		"Waits up to seconds (default 20, at most 120) for one when none is waiting. " +
		"Loop on listen to be the player's helper in game.",
		object(schema{"seconds": schema{"type": "integer", "minimum": 0, "maximum": 120}})},
	{"goto", "Flies your avatar to x,y,z (its feet). Starts a job.", point},
	{"clear", "Breaks every breakable block in the box, top layer first, with the biggest brushes that fit. " +
		"At most 32,768 blocks. Starts a job.",
		object(box, "from", "to")},
	{"fill", "Fills the box's empty cells (air, water, plants) with block, bottom layer first; " +
		"block \"air\" clears the box instead, e.g. for a doorway. " +
		"At most 32,768 blocks. Uses the biggest brushes that fit; sides that are multiples of 3 go fastest. Starts a job.",
		object(with(box, schema{"block": schema{"type": "string"}}), "from", "to", "block")},
	{"place", "Sets each listed cell to its block; \"air\" breaks what is there. Up to 4,096 cells, " +
		"so it can build any design. Cells of one block that form cubes are done with one brush each, " +
		"but fill is quicker for solid boxes. Starts a job.",
		object(schema{"blocks": schema{"type": "array", "minItems": 1, "maxItems": MaxPlace,
			"items": object(with(point["properties"].(schema), schema{"block": schema{"type": "string"}}), "x", "y", "z", "block")}},
			"blocks")},
	{"build", "Builds a template with its front's bottom middle at x,y,z, facing north, east, south or west " +
		"(default south), the rest behind it. Templates, with their default width, depth and height: " +
		"cabin 9×9×13 (log, with a doorway, windows and a planks gable roof), tower 9×9×21 (cobblestone, " +
		"with a doorway, window slits and battlements), wall 9×1×6 (stone), floor 9×9×1 (planks) and " +
		"stairs 3×9×9 (cobblestone, one block up per step). w, d and h change the size, up to 32 each; " +
		"material changes the main block. It clears what's in its way, inside included. " +
		"template may also name a saved plan, which can't be resized. Starts a job.",
		object(with(point["properties"].(schema), schema{
			"template": schema{"type": "string"},
			"w":        schema{"type": "integer", "minimum": 1, "maximum": 32},
			"d":        schema{"type": "integer", "minimum": 1, "maximum": 32},
			"h":        schema{"type": "integer", "minimum": 1, "maximum": 32},
			"material": schema{"type": "string"},
			"facing":   facing,
		}), "template", "x", "y", "z")},
	{"propose", "Shows the player a plan as a ghost at x,y,z, its front facing north, east, south or west " +
		"(default south), and waits for their /yes to build it, or /no. plan is a whole plan " +
		"{name, about?, palette, layers} or the name of a saved one. material swaps its most-used block. " +
		"A new propose replaces the ghost. Starts a job.",
		object(with(point["properties"].(schema), schema{
			"plan":     planInput,
			"facing":   facing,
			"material": schema{"type": "string"},
		}), "plan", "x", "y", "z")},
	{"save_plan", "Saves the plan last proposed as one of the player's plans, under name or its own name. " +
		"Then build or /build can build it by name.",
		object(schema{"name": planName})},
	{"plans", "Lists the player's plans, then the ones others share: name, about, size, cells, owner.", noArgs},
	{"plan", "One saved plan in full, with its palette and layers, to build on or change.",
		object(schema{"name": planName}, "name")},
	{"chat", "Lines said in chat after the id after (all kept lines, up to 20, when left out), oldest first, " +
		"with ids, who said them, how many seconds ago, and agent: true for lines said with say; last is the newest id. " +
		"wait: true first waits up to 20 seconds for a player to say something after that id. " +
		"Loop on chat with after: last and wait: true to listen for the player's answer.",
		object(schema{"after": schema{"type": "integer", "minimum": 0}, "wait": schema{"type": "boolean"}})},
	{"stop", "Stops the current job.", noArgs},
	{"undo", "Puts back what the last job changed. Starts a job.", noArgs},
}

// ToolNamed finds a tool by name.
func ToolNamed(name string) (Tool, bool) {
	for _, t := range Tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}
