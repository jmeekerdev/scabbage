// Package inventory holds the blocks a survival player carries.
package inventory

import (
	"encoding/binary"
	"errors"

	"scabbage/internal/world"
)

const (
	Slots    = 36 // the first Hotbar slots are the hotbar
	Hotbar   = 9
	StackMax = 64
)

// Stack is some number of one block; the zero Stack is an empty slot.
type Stack struct {
	Block world.Block
	Count uint8
}

type Inventory [Slots]Stack

// Add puts n of b into matching stacks first, then empty slots, and returns how many did not fit.
func (inv *Inventory) Add(b world.Block, n int) int {
	for i := range inv {
		if s := &inv[i]; s.Block == b && s.Count > 0 {
			n = s.fill(n)
		}
	}
	for i := range inv {
		if s := &inv[i]; s.Count == 0 && n > 0 {
			s.Block = b
			n = s.fill(n)
		}
	}
	return n
}

// fill tops up the stack from n and returns what is left of n.
func (s *Stack) fill(n int) int {
	room := min(n, StackMax-int(s.Count))
	s.Count += uint8(room)
	return n - room
}

// Take removes one b from slot, reporting whether the slot held one.
func (inv *Inventory) Take(slot int, b world.Block) bool {
	if slot < 0 || slot >= Slots || inv[slot].Block != b || inv[slot].Count == 0 {
		return false
	}
	if inv[slot].Count--; inv[slot].Count == 0 {
		inv[slot] = Stack{}
	}
	return true
}

// Move drops the stack at from onto to: matching blocks merge, anything else swaps.
func (inv *Inventory) Move(from, to int) bool {
	if from == to || from < 0 || to < 0 || from >= Slots || to >= Slots {
		return false
	}
	src, dst := &inv[from], &inv[to]
	if src.Block == dst.Block && dst.Count > 0 {
		left := dst.fill(int(src.Count))
		if src.Count = uint8(left); left == 0 {
			*src = Stack{}
		}
		return true
	}
	*src, *dst = *dst, *src
	return true
}

const slotBytes = 3 // block uint16 then count

var ErrCorrupt = errors.New("inventory: wrong saved size")

// Encode packs the inventory for storage.
func (inv *Inventory) Encode() []byte {
	out := make([]byte, 0, Slots*slotBytes)
	for _, s := range inv {
		out = binary.LittleEndian.AppendUint16(out, uint16(s.Block))
		out = append(out, s.Count)
	}
	return out
}

// Decode unpacks a stored inventory; blocks that no longer exist vanish.
func Decode(data []byte) (Inventory, error) {
	var inv Inventory
	if len(data) == 0 {
		return inv, nil
	}
	if len(data) != Slots*slotBytes {
		return inv, ErrCorrupt
	}
	for i := range inv {
		b := world.Block(binary.LittleEndian.Uint16(data[i*slotBytes:]))
		if n := data[i*slotBytes+2]; n > 0 && b != world.Air && b.Valid() {
			inv[i] = Stack{b, min(n, StackMax)}
		}
	}
	return inv, nil
}
