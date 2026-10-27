import { typeIconComponents } from "./type-icons";

export function dropdownPath(id: string, options: { id: string; name: string; parent?: string }[]) {
  const parts: string[] = [];
  const seen = new Set<string>();
  let current = options.find((option) => option.id === id);
  while (current && !seen.has(current.id)) {
    seen.add(current.id);
    parts.unshift(current.name);
    current = options.find((option) => option.id === current?.parent);
  }
  return parts.join(" → ");
}
export function OptionIcon({ icon }: { icon: string }) {
  const Icon = Object.hasOwn(typeIconComponents, icon) ? typeIconComponents[icon] : undefined;
  return Icon ? <Icon aria-hidden className="mr-1 inline size-3.5" /> : null;
}
