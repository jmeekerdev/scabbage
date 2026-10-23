interface Command {
  help: string;
  run(arg: string): void;
  choices?: () => readonly string[]; // what the last argument may be, for completion
}

/** Slash commands the client handles itself, such as "/settings", with completion for typing. */
export class Commands {
  private readonly own = new Map<string, Command>();
  private readonly others = new Set<string>();

  add(
    name: string,
    help: string,
    run: (arg: string) => void,
    choices?: () => readonly string[],
  ): void {
    this.own.set(name, { help, run, choices });
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

  /** What fits what is typed: command names while the name is typed, then argument choices. */
  complete(text: string): string[] {
    const name = /^\/(\S*)$/.exec(text);
    if (name) return this.names(name[1].toLowerCase()).map((n) => `/${n}`);
    const arg = /^\/(\S+)\s.*?(\S+)$/.exec(text);
    const choices = arg && this.own.get(arg[1].toLowerCase())?.choices?.();
    if (!choices) return [];
    const typed = arg[2].toLowerCase();
    return choices.filter((c) => c.startsWith(typed)).sort();
  }

  /** `text` with the command name or last argument filled in once only one fits. */
  finish(text: string): string {
    const fits = this.complete(text);
    if (fits.length !== 1) return text;
    return fits[0].startsWith("/") ? `${fits[0]} ` : `${text.replace(/\S+$/, fits[0])} `;
  }

  private names(typed: string): string[] {
    return [...new Set([...this.own.keys(), ...this.others])]
      .filter((n) => n.startsWith(typed))
      .sort();
  }

  help(): string[] {
    return [...this.own.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([name, c]) => `/${name}: ${c.help}`);
  }
}
