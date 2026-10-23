# Building

## The world

- **Size.** Blocks are a third of a meter: 3 blocks make a meter. The world is about 1 km
  across (a border 512 m from the middle) and 128 m tall. The border and the floor are walls;
  the sky is open.
- **Land.** Every world grows from a seed: rolling hills, stepped shores and lakes, rocky crags,
  meadows with flowers, oak, pine and slim trees, bushes and boulders. Deep below are
  [dungeons](adventure.md#dungeons).
- **Spawn** is in the middle, at x 0, z 0. Around it is a zone only admins can build in.
- **Time and weather** are the same for everyone. A day lasts 20 minutes. Every 4 minutes the
  weather changes: usually clear, sometimes rain, now and then a storm. High up it snows.
- **Coordinates.** The 📍 line on screen shows where your feet are (x, y, z) and which way you
  face. y is up, and north is −z.

## Breaking and placing

- Left-click breaks and right-click places. The brush (G) sets how big: 3, 2 or 1 blocks wide.
  The 3-wide brush snaps to a grid, so walls line up.
- You can reach about 5 m.
- You can't place a solid block where someone is standing, and nobody can break or place
  bedrock.
- Breaking the block under a flower or tuft of grass removes the plant too.
- **Zones** keep building safe. Inside a zone only its builders and admins can change blocks.
  Outside zones, anyone can. If a block won't change, you're probably in a zone. Ask an admin
  to add you as a builder.

## Blocks

Open **Blocks** (B) for the full list. Some highlights:

| Kind              | Blocks                                                                                 |
| ----------------- | -------------------------------------------------------------------------------------- |
| Ground            | grass, dirt, sand, gravel, stone, cobblestone, bedrock                                 |
| Building          | planks, brick, glass, log, leaves, pine leaves                                         |
| Shapes            | slabs and stairs in stone, cobblestone, planks and brick; stairs face the way you look |
| Wool              | 16 colors                                                                              |
| Light             | lamp, torch. Light reaches about 10 m                                                  |
| Plants            | tall grass, red, yellow, blue and white flowers. You can walk through them             |
| Circuits          | wire, switches, doors, pistons and many more: see [Logic](logic.md)                    |
| Games and stories | portal, checkpoint, teleporter, script block, chest, boss altar, gold and gem blocks   |

## Paint your own blocks

Type `/create` (or press 🎨) and choose **🧱 New block** to paint a 16×16 block with Draw, Fill,
Erase and Pick color. Name it (up to 20 characters) and send it. An admin approves it, and then
it's in everyone's Blocks menu. Admins' blocks are ready at once. Transparent pixels make a
see-through block.

The same way, **🧑 My look** paints your avatar's 64×32 skin with a live 3D preview.

A world holds up to 64 painted blocks. Each player can have 3 waiting for approval. Admins can
also repaint built-in blocks.

## Autopilot

In creative, the autopilot flies and builds for you:

| Command                         | Does                                                                                                  |
| ------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `/goto X Y Z`                   | flies there. `~` means where you are now, and `~5` means 5 more, so `/goto ~ ~20 ~` goes up 20 blocks |
| `/fill X1 Y1 Z1 X2 Y2 Z2 BLOCK` | fills the empty spots in the box, bottom layer first. Tab completes block names                       |
| `/clear X1 Y1 Z1 X2 Y2 Z2`      | breaks every block in the box, top layer first                                                        |
| `/build NAME`                   | builds a template just in front of you. See [Templates](#templates)                                   |
| `/yes`, `/no`                   | build or cancel the job waiting for you                                                               |
| `/stop`                         | stops                                                                                                 |
| `/undo`                         | takes back the last job. Undo again to redo it                                                        |

Corners are both included, and a job can be up to 32,768 blocks (a 32-block cube). The
autopilot uses the biggest brushes that fit, about 8 a second, so boxes with sides that are
multiples of 3 go fastest. It follows your rules: a job that reaches into a zone you can't
build in is refused. Moving or building yourself stops it.

A job that changes more than 500 blocks waits for you first. The blocks it will build show as
see-through blue and the blocks it will break as red, and chat asks "Build it? /yes builds, /no
cancels." You can fly around the preview before you answer.

### Templates

`/build NAME [W D H] [BLOCK] [FACING] [at X Y Z]` builds one of these:

| Template | Default size (W×D×H) | Made of                                                                   |
| -------- | -------------------- | ------------------------------------------------------------------------- |
| `cabin`  | 9×9×13               | log walls with a doorway and glass windows, a planks floor and gable roof |
| `tower`  | 9×9×21               | cobblestone, with a doorway, window slits, a flat top and battlements     |
| `wall`   | 9×1×6                | stone                                                                     |
| `floor`  | 9×9×1                | planks                                                                    |
| `stairs` | 3×9×9                | cobblestone steps, one block up per step, with the space above cleared    |

W is the width across the front, D the depth and H the height, up to 32 each. BLOCK swaps the
main material, so `/build cabin brick` makes a brick cabin with the same windows and roof.

The front's bottom middle goes 3 blocks in front of your feet, at your feet's height, and the
rest goes away from you, with the door facing you. FACING (`north`, `east`, `south` or
`west`) turns it, and `at X Y Z` puts the front somewhere else, with `~` for where you are. A
template clears what's in its way, inside included, so you can build into a hill.

An [agent](agents.md) uses the same autopilot and templates.
