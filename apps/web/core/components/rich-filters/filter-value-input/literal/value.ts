import type { TLiteralFilterFieldConfig, TFilterValue, SingleOrArray } from "@plane/types";

export type LiteralDraft = { start: string; end: string };
export type LiteralValue = string | number | number[];
type ParsedValue = { value: LiteralValue; error?: never } | { error: string; value?: never };

export function literalDraft(value: SingleOrArray<TFilterValue>): LiteralDraft {
  const values = Array.isArray(value) ? value : [value];
  return { start: formatLiteral(values[0]), end: formatLiteral(values[1]) };
}

function formatLiteral(value: TFilterValue): string {
  return typeof value === "string" || typeof value === "number" ? String(value) : "";
}

function parseNumber(value: string): number | undefined {
  const text = value.trim();
  if (!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i.test(text)) return undefined;
  const number = Number(text);
  return Number.isFinite(number) ? number : undefined;
}

export function parseLiteralValue(kind: TLiteralFilterFieldConfig["type"], draft: LiteralDraft): ParsedValue {
  if (kind === "text") return { value: draft.start };
  const start = parseNumber(draft.start);
  if (start === undefined) return { error: "Enter a finite number." };
  if (kind === "number") return { value: start };
  const end = parseNumber(draft.end);
  if (end === undefined) return { error: "Enter a finite number for both ends of the range." };
  if (start > end) return { error: "The lower bound must not exceed the upper bound." };
  return { value: [start, end] };
}

export function literalLabel(value: SingleOrArray<TFilterValue>): string {
  const draft = literalDraft(value);
  if (Array.isArray(value)) return `${draft.start || "—"} – ${draft.end || "—"}`;
  return draft.start || "—";
}
