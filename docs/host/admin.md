# Running a server

Admins let people in, choose who plays creative or survival, protect builds with zones, and
set up minigames and adventures. The first admin joins with the code the server logs at first
start (see [Install](install.md)).

## Letting people in

1. Type `/invite` (or use the admin page) to make an invite link. Choose how many people it's
   for (1, 5 or 10) and how long it lasts (an hour, a day or a week).
2. Send the link. When someone opens it and asks to join, a toast appears for every admin: press
   **Let in** or **No**. You can also type `/approve NAME` or `/deny NAME`.

Requests wait up to 24 hours, and up to 10 can wait at once. A request with a name that's
already taken links the new device to that player. The admin page flags these: approve them
only if it's really that person.

Nobody can play without an admin's approval, and a link alone isn't enough.

## Admins

`/admin NAME` makes someone an admin and `/unadmin NAME` takes it away. You can't change
yourself, so there's always someone who can undo it. Admins:

- build anywhere, including the spawn zone and every other zone;
- approve players and painted blocks and skins (their own are approved at once);
- use the admin commands below, and can repaint built-in blocks;
- can link agents for themselves without being allowed first.

## Creative and survival

- The server's `SCABBAGE_MODE` is the mode for everyone who hasn't been given one.
- `/creative NAME` and `/survival NAME` set a player's own mode.
- `/zone mode ZONE creative|survival|any` makes a zone override everyone's mode while they're
  in it. `any` leaves players in their own.

## Zones

A zone is a square of land, from bedrock to sky, that only its builders and admins can change.
Outside zones, anyone can build. Where zones overlap, the smallest one counts.

| Command                                   | Does                                                              |
| ----------------------------------------- | ----------------------------------------------------------------- |
| `/zone add NAME SIZE`                     | makes a square zone SIZE blocks wide (up to 768), centered on you |
| `/zone allow NAME PLAYER`                 | lets a player build there                                         |
| `/zone disallow NAME PLAYER`              | stops them                                                        |
| `/zone mode NAME creative\|survival\|any` | sets the zone's game mode                                         |
| `/zone remove NAME`                       | removes the zone (the blocks stay)                                |
| `/zone list`                              | lists zones                                                       |

Names are up to 24 letters, and capitals don't matter. Players see a greeting when they walk
into a zone.

The **Spawn** zone is built in: admins only, `SCABBAGE_SPAWN_ZONE` blocks out from the middle
in each direction. Save your own zone called `Spawn` to change it.

Zones also hold [arenas](../play/minigames.md) and [adventure maps](../play/adventure.md#adventure-maps).

## Getting around

`/tp X Z` drops you on top of that column, `/tp X Y Z` takes you to an exact spot, and
`/tp NAME` takes you to a player.

## Painted blocks and skins

When a player sends a block or skin from `/create`, admins get a toast to approve it (**Looks
good**) or not. Approved blocks join everyone's Blocks menu, and approved skins show to everyone.
A world holds 64 painted blocks. Admins can also repaint built-in blocks from Create.

## Agents

Players need an admin's go-ahead before they can link an AI agent: `/agent allow NAME`, and
`/agent deny NAME` to turn it off and revoke their tokens. See [Agents](../play/agents.md).

## Starting the world over

`/world reset` starts the world over from its seed, and `/world new [SEED]` makes a new one. A
second admin has to approve with `/world approve CODE` within 10 minutes. If you're the only
admin, you approve it yourself. Any admin can cancel with `/world deny`.

This removes every built block, zone, map, script and painted block. Players, their backpacks
and quests stay. The database is backed up to `backups/before-wipe-….db` first.

## The admin page

Open `/admin.html` on a device where you're an admin. It shows:

- **Waiting to join:** requests to approve or deny.
- **New art:** painted blocks and skins to approve.
- **Invite:** make a link to copy or share.
- **Players:** make anyone creative or survival, or an admin.

Opening the admin page doesn't load the world, so it's cheap to leave open.
