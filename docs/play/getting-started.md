# Getting started

## Join

1. Open the invite link an admin sent you. It looks like `https://game.example.org/#join=ABCD-EFGH`.
   You can also open the game and type the code; capitals and dashes don't matter.
2. Pick a name (1–16 letters, digits, spaces, `_` or `-`) and a color, then press **Ask to join**.
3. Wait on the "Almost in!" screen while an admin lets you in. Keep the page open.

From then on, this browser remembers you, and opening the game takes you straight in. The
browser keeps a secret key for you in its storage. If you clear the site's data or switch
browsers, you'll need a new invite. An admin can link the new device to your old player by
letting in a request with the same name.

If you type a wrong code five times, you'll have to wait 10 minutes before trying again.

## Controls

Click the game to take the mouse. Esc gives it back.

| Key            | Does                                                                                          |
| -------------- | --------------------------------------------------------------------------------------------- |
| WASD or arrows | walk                                                                                          |
| Space          | jump. Double-tap it, or press F, to fly (creative only)                                       |
| Shift          | go down while flying. Shift + right-click builds against a door or switch instead of using it |
| Mouse          | look                                                                                          |
| Left click     | break. In survival, hold it to dig. Also hits animals and monsters                            |
| Right click    | place, use a switch or door, or talk to a character                                           |
| Middle click   | pick the block you're looking at                                                              |
| Wheel or 1–9   | choose a hotbar slot                                                                          |
| G              | change the brush: 1, 2 or 3 blocks wide                                                       |
| B              | Blocks (Backpack in survival)                                                                 |
| Q              | Quests                                                                                        |
| E (hold)       | emote wheel: point and let go                                                                 |
| R              | read or write the script in a script block                                                    |
| Enter, T or /  | chat and commands. Tab completes them                                                         |

Holding a mouse button repeats every 0.16 s.

**Gamepad.** The left stick walks and the right stick looks. A jumps; double-tap A or press Y to fly, and
B goes down. RT breaks and LT places. X picks a block. LB and RB change the hotbar slot, and
clicking the right stick changes the brush. Back opens a script. The d-pad sends emotes: up 👋,
right 👍, down 😢, left 🎉. Gamepads need HTTPS in most browsers.

Phones and tablets have touch controls, but they aren't a target right now.

## Brushes and scale

Blocks are small: **3 blocks make a meter**, and you're about 5 blocks tall. You build with a
brush, a cube 3, 2 or 1 blocks wide. Press **G** to change it. The 3-wide brush builds a
meter-sized block at a time. The thin ones are for details.

## Menus

Open menus with their keys, by typing their name as a command (`/settings`), or with on-screen
buttons (`/buttons` shows or hides them).

- **Settings:** view distance, the [look](looks.md) of the world, graphics quality, a frame
  rate limit, automatic quality, sound effects, music and the menu buttons. Settings are kept in
  this browser.
- **Blocks (B):** in creative, every block to put in your hotbar. In survival this is your
  **Backpack** and crafting. See [Survival](survival.md).
- **Quests (Q):** your quests, newest first. See [Adventure](adventure.md).
- **Create (`/create`):** paint your own skin (🧑 My look) or a new block (🧱 New block).
  See [Building](building.md#paint-your-own-blocks).
- **Emotes (E or `/emotes`):** 👋 Wave, 😀 Happy, 😂 Laugh, ❤️ Love, 👍 Thumbs up, 😮 Wow,
  😢 Sad, 🎉 Party. They pop up over your head for everyone.

## Chat

Press Enter or T, type and press Enter. Lines can be up to 200 characters. Chat, emotes and
commands share a limit of about one message every two seconds, with a burst of 5. Go faster and
you'll see "Slow down a little!".

`/help` lists the commands you can use. They're all in [Commands](../host/commands.md).

## Pausing

When another window has focus, the game pauses, goes quiet and shows **⏸ Paused**. Others see
💤 by your name, and nothing can hurt you while you're away. It carries on when you come back.
While an [agent](agents.md) is building for you, the game keeps running.

## Next

- [Building](building.md): blocks, brushes, painting blocks and the autopilot.
- [Survival](survival.md): health, digging, crafting and monsters.
- [Logic](logic.md): wire, switches, gates and machines.
- [Minigames](minigames.md): races, capture the flag, tag and more.
- [Scripting](scripting.md): Blockly and Lua.
- [Adventure](adventure.md): characters, quests, dungeons and adventure maps.
- [Agents](agents.md): let an AI agent build as you.
