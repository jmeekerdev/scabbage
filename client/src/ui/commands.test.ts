import { describe, expect, it } from "vitest";
import { Commands } from "./commands";

function setup() {
  const ran: string[] = [];
  const commands = new Commands();
  commands.add("settings", "change graphics and sound", () => ran.push("settings"));
  commands.add("say", "repeat", (arg) => ran.push(`say:${arg}`));
  commands.add("blocks", "choose blocks", () => ran.push("blocks"));
  return { commands, ran };
}

describe("client commands", () => {
  it("run by name, ignoring case and spaces around", () => {
    const { commands, ran } = setup();
    expect(commands.run("  /Settings ")).toBe(true);
    expect(ran).toEqual(["settings"]);
  });

  it("pass the rest of the line as the argument", () => {
    const { commands, ran } = setup();
    commands.run("/say hello  there");
    expect(ran).toEqual(["say:hello  there"]);
  });

  it("leave chat and unknown commands for the server", () => {
    const { commands, ran } = setup();
    expect(commands.run("settings")).toBe(false);
    expect(commands.run("/tp 1 2 3")).toBe(false);
    expect(commands.run("/")).toBe(false);
    expect(ran).toEqual([]);
  });

  it("complete names from what is typed, with server ones too", () => {
    const { commands } = setup();
    commands.also(["tp", "leave"]);
    expect(commands.complete("/s")).toEqual(["/say", "/settings"]);
    expect(commands.complete("/")).toEqual(["/blocks", "/leave", "/say", "/settings", "/tp"]);
    expect(commands.complete("/settings x")).toEqual([]);
    expect(commands.complete("hi")).toEqual([]);
  });

  it("finish a name once only one matches", () => {
    const { commands } = setup();
    expect(commands.finish("/se")).toBe("/settings ");
    expect(commands.finish("/s")).toBe("/s");
    expect(commands.finish("/x")).toBe("/x");
  });

  it("complete a command's last argument from its own choices", () => {
    const { commands } = setup();
    commands.add(
      "fill",
      "fill a box",
      () => undefined,
      () => ["stone", "stone_stairs", "planks"],
    );
    expect(commands.complete("/fill 1 2 3 4 5 6 st")).toEqual(["stone", "stone_stairs"]);
    expect(commands.complete("/fill 1 2 3 4 5 6 ")).toEqual([]);
    expect(commands.complete("/say st")).toEqual([]);
    expect(commands.finish("/fill 1 2 3 4 5 6 pl")).toBe("/fill 1 2 3 4 5 6 planks ");
    expect(commands.finish("/fill 1 2 3 4 5 6 st")).toBe("/fill 1 2 3 4 5 6 st");
  });

  it("list themselves for help", () => {
    const { commands } = setup();
    expect(commands.help()).toEqual([
      "/blocks: choose blocks",
      "/say: repeat",
      "/settings: change graphics and sound",
    ]);
  });
});
