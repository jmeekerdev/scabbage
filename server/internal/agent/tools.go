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
goto, clear, fill, place and undo start a job and return at once; then call status with wait: true
until its state isn't running (each wait lasts up to 20 seconds). One job runs at a time, and the
player stops it by moving. The avatar flies to the work itself, over the trees when it finds no way
through, so there's no need to goto first. Edits follow the player's own rules.
Every job really changes the world wherever its cells are, even high in the sky: there's no dry
run, so build only where you mean to. undo takes back the last job.`

type schema = map[string]any

var (
	integer = schema{"type": "integer"}
	point   = object(schema{"x": integer, "y": integer, "z": integer}, "x", "y", "z")
	box     = schema{"from": point, "to": point}
	noArgs  = object(schema{})
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
	{"say", "Says something in chat as the player, marked 🤖 so everyone knows an agent said it.",
		object(schema{"text": schema{"type": "string", "minLength": 1, "maxLength": 200}}, "text")},
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
