# Scripting

A **script block** runs a small program when something happens: someone uses it, its power
changes, someone talks to its character, or a round starts or ends. Write scripts with
**Blockly** blocks you snap together, or in **Lua**. Both do the same things.

## Write a script

1. Place a `script_block`. In survival you can [craft one](survival.md#crafting).
2. Look at it and press **R** (Back on a gamepad).
3. Build the script in **🧩 Blocks**, or type it in **⌨️ Lua**, then press **Save**.

You can save a script wherever you can build. Anyone can read one, but outside your zones it's
read-only. A script acts with the rights of whoever saved it last. Breaking the block deletes
its script.

A script made with blocks can be opened as Lua to see what it does. Going back to blocks after
changing the Lua loses those changes.

If something goes wrong, the editor shows the error and the line, and chat tells the script's
owner and whoever set it off: "The script block at X Y Z went wrong: …". A script with an error
is still saved, so you can fix it.

## Blockly

| Category                                              | Blocks                                                                                                                             |
| ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| **When**                                              | someone uses this block · someone talks to me · power changes · a round starts · a round ends                                      |
| **Story**                                             | I am a character called … · tell · ask (up to 3 choices) · answer · give a quest · finish a quest · remember · recall · take items |
| **World**                                             | say · place a block at a spot east/up/south of this block · after N seconds                                                        |
| **Players**                                           | player · power is on · players in the round · winners · give items · teleport · give points                                        |
| Logic, Loops, Math, Text, Lists, Variables, Functions | the usual Blockly blocks                                                                                                           |

Blocks outside a **When** block are grayed out and don't run.

## Lua

The top of the script runs when it's saved or loaded. Define these functions to answer events:

| Event                     | When                                  |
| ------------------------- | ------------------------------------- |
| `on_touch(player)`        | someone uses (right-clicks) the block |
| `on_power(on)`            | its power turns on or off             |
| `on_talk(player)`         | someone talks to its character        |
| `on_round_start(players)` | a round starts in the arena it's in   |
| `on_round_end(winners)`   | that round ends                       |

`player` is a player's name. `here.x`, `here.y` and `here.z` are the block's position.

| Call                               | Does                                                                                                                                         |
| ---------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `say(text)`                        | chats as "Script" to everyone within 32 m                                                                                                    |
| `set_block(x, y, z, name)`         | places a block (or `"air"`) within 16 m, where the owner can build. Not bedrock or script blocks, and not a solid block where someone stands |
| `give_item(player, item, count)`   | gives 1–64 of an item (count defaults to 1)                                                                                                  |
| `take_item(player, item, count)`   | takes items. Returns `true` if they had them                                                                                                 |
| `teleport(player, x, y, z)`        | moves a player near the block anywhere in the world                                                                                          |
| `add_points(player, n)`            | gives or takes up to 10 points during a round. See [Minigames](minigames.md)                                                                 |
| `after(seconds, fn)`               | runs `fn` once later, from 0.1 s to an hour. Up to 16 at a time                                                                              |
| `npc(name)`                        | stands a character on top of the block                                                                                                       |
| `talk(player, text)`               | shows a message from the character                                                                                                           |
| `ask(player, text, {choices}, fn)` | asks a question with 1–4 choices; `fn(answer)` gets the one they picked                                                                      |
| `remember(player, key, value)`     | keeps a string, number or true/false about a player; `nil` forgets it. Any script can read it                                                |
| `recall(player, key)`              | reads it back, or `nil`                                                                                                                      |
| `quest(player, title, step)`       | starts a quest or moves it to its next step                                                                                                  |
| `finish_quest(player, title)`      | marks a quest done                                                                                                                           |

Players have to be within 16 m of the block for calls about them. A call that isn't allowed
raises an error you can catch with `pcall`.

Only scripts saved by an admin give items to, or take them from, survival players. In creative
`take_item` always says yes and takes nothing.

```lua
npc("Granny Ivy")

function on_talk(player)
  if recall(player, "met") then
    talk(player, "Back again? The well is to the north.")
    return
  end
  ask(player, "Will you find my lost sheep?", {"Yes!", "Not now"}, function(answer)
    if answer == "Yes!" then
      remember(player, "met", true)
      quest(player, "Lost sheep", "Look for a sheep by the well")
    end
  end)
end

function on_power(on)
  if on then set_block(here.x, here.y + 3, here.z, "lamp") end
end
```

## Limits

Scripts run in a sandbox on the server, so they can't slow the game down or reach outside it.

- Scripts can be up to 16 KB.
- Each run gets 10 ms. A script that takes longer is stopped until it's saved again.
- One run can make 64 calls.
- One event can set off at most 32 scripts, through the blocks they change.
- The Lua `string`, `table` and `math` libraries work. Loading code, files, `print` and
  `string.rep` don't.
- Timers and open questions are forgotten when the server restarts. Characters, memories and
  quests are kept.
