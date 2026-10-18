type PropValue = string | number | boolean | ((ev: Event) => void) | undefined;
type Child = Node | string | false | undefined;

/** Creates an element: `on*` props become listeners, booleans toggle attributes, the rest are attributes. */
export function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  props: Record<string, PropValue> = {},
  ...children: Child[]
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(props)) {
    if (typeof value === "function") node.addEventListener(key.slice(2), value);
    else if (typeof value === "boolean") node.toggleAttribute(key, value);
    else if (value !== undefined) node.setAttribute(key, String(value));
  }
  for (const child of children) if (child) node.append(child);
  return node;
}

export function $<T extends HTMLElement>(selector: string): T {
  const node = document.querySelector<T>(selector);
  if (!node) throw new Error(`missing element ${selector}`);
  return node;
}
