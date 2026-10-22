interface Command {
  help: string;
  run(arg: string): void;
}

/** Slash commands the client handles itself, such as "/settings", with completion for typing. */
export class Commands {
  private readonly own = new Map<string, Command>();
  private readonly others = new Set<string>();

  add(name: string, help: string, run: (arg: string) => void): void {
    this.own.set(name, { help, run });
  }

  /** Names the server answers, offered when completing but left for the server to run. */
  also(names: readonly string[]): void {
    for (const n of names) this.others.add(n);
  }

  /** Runs `text` if it is one of these commands, and says whether it was. */
  run(text: string): boolean {
    const m = /^\/(\S+)\s*(.*)$/.exec(text.trim());
    const command = m && this.own.get(m[1].toLowerCase());
    if (!command) return false;
    command.run(m[2]);
    return true;
  }

  /** Command names starting with what is typed, while only the name is typed. */
  complete(text: string): string[] {
    const m = /^\/(\S*)$/.exec(text);
    if (!m) return [];
    const typed = m[1].toLowerCase();
    return [...new Set([...this.own.keys(), ...this.others])]
      .filter((n) => n.startsWith(typed))
      .sort();
  }

  /** `text` with the command name filled in once only one name fits. */
  finish(text: string): string {
    const names = this.complete(text);
    return names.length === 1 ? `/${names[0]} ` : text;
  }

  help(): string[] {
    return [...this.own.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([name, c]) => `/${name}: ${c.help}`);
  }
}
