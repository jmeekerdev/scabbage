# Survival

In survival you can't fly, blocks take time to dig, and you build with what you've collected.
Admins choose who plays survival: for each player, for each zone, or for the whole server.
A zone's mode wins while you stand in it. Switching modes heals you.

## Health

- You have 10 hearts (20 points). After 5 seconds without getting hurt, you heal a point every
  3 seconds. There's no hunger.
- **Falls** up to 3 m are safe. After that, each meter costs a point, so a 10 m fall costs 7.
  Landing in water is safe.
- **Monsters** bite: a slime takes a heart, King Slime two, the Stone Golem three.
- **Other players** can only hit you in a [minigame arena](minigames.md#pvp) with PvP on.
- **Fainting.** At zero you faint and come back with full health, keeping your backpack.
  You come back at your last [checkpoint](logic.md#teleporters-and-checkpoints) if it's still
  there and in the same zone, and otherwise at spawn.

## Digging and items

- Hold left-click to dig. Bigger brushes take longer: 3 wide of stone takes about 1.5 s. Leaves
  and glass go fast, stone and brick slowly. There are no tools to make.
- What you dig drops on the ground. Walk over it to pick it up. Stone drops cobblestone and
  grass drops dirt. Drops vanish after 5 minutes.
- Placing a block uses one from your hotbar slot.

## Backpack

Press **B**. The backpack has 36 slots, and the top row of 9 is your hotbar. Tap an item, then a
slot, to move it. A stack holds up to 999.

## Crafting

The **Make** section of the backpack shows what you can make from what you carry. Ingredients
don't need arranging.

| Make                                   | From                                                  |
| -------------------------------------- | ----------------------------------------------------- |
| 4 planks                               | 1 log                                                 |
| 4 torches                              | 1 planks                                              |
| glass                                  | sand                                                  |
| stone                                  | cobblestone                                           |
| 2 brick                                | dirt + sand                                           |
| lamp                                   | glass + torch                                         |
| white wool                             | 4 leaves                                              |
| yellow, brown, green, gray or red wool | white wool + sand, dirt, leaves, cobblestone or brick |
| 6 slabs                                | 3 stone, cobblestone, planks or brick                 |
| 4 stairs                               | 6 stone, cobblestone, planks or brick                 |
| 8 wire                                 | torch + cobblestone                                   |
| switch                                 | planks + cobblestone                                  |
| button                                 | stone                                                 |
| 2 pressure plates                      | 2 stone                                               |
| signal lamp                            | lamp + wire                                           |
| door                                   | 2 planks                                              |
| piston                                 | planks + 2 cobblestone + wire                         |
| timer                                  | stone + torch + wire                                  |
| counter                                | stone + glass + wire                                  |
| 2 teleporters                          | 2 glass + lamp + 2 wire                               |
| portal                                 | teleporter + lamp                                     |
| note block                             | planks + wire                                         |
| script block                           | glass + lamp + 2 wire                                 |

Hosts can change the recipes: see [Install](../host/install.md#recipes).

## Animals and monsters

|             | Health | Speed   | Notes                                                                               |
| ----------- | ------ | ------- | ----------------------------------------------------------------------------------- |
| Sheep       | 1      | slow    | wander on grass by day. Hit one to shear it for white wool; it regrows in 3 minutes |
| Slime       | 3      | 2.5 m/s | come out at night and chase you from up to 16 m. They melt in daylight              |
| King Slime  | 24     | 2 m/s   | a dungeon boss. Lets out a slime each time it loses 6 health                        |
| Stone Golem | 40     | 1.6 m/s | a dungeon boss. Hits hard and doesn't flinch                                        |

Monsters only go after survival players who aren't away. When everyone's game is paused, the
animals and monsters stop too. Left-click hits them, from up to 5.5 m.

Bosses and treasure are in [Adventure](adventure.md).
