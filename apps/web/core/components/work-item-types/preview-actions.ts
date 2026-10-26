import { resolvePropertyContext } from "./context-resolution";
import sortBy from "lodash-es/sortBy.js";
import { convertHTMLToMarkdown } from "@plane/utils/editor/markdown-parser";
import type { PreviewContent, PreviewItem, PreviewProperty } from "./preview-content";

function valueLabel(property: PreviewProperty, value: PreviewItem["values"][string]) {
  if (property.kind === "Rich text" && typeof value === "string")
    return convertHTMLToMarkdown({ description_html: value, metaData: { file_assets: [], user_mentions: [] } }).trim();
  const label = (id: string) => property.options.find((option) => option.id === id)?.name ?? id;
  return Array.isArray(value)
    ? value.map(label).join(" / ")
    : typeof value === "boolean"
      ? value
        ? "Yes"
        : "No"
      : label(value);
}
export function preserveProperties(item: PreviewItem, properties: PreviewProperty[], reason: string): PreviewItem {
  const values = { ...item.values };
  const removed = properties.filter((property) => values[property.id] !== undefined && values[property.id] !== "");
  if (!removed.length) return item;
  const text = removed.map((property) => `${property.name}: ${valueLabel(property, values[property.id])}`).join("\n");
  removed.forEach((property) => {
    delete values[property.id];
  });
  return {
    ...item,
    values,
    description: `${item.description}\n\n${reason}\n${text}`,
    activity: [`${reason}. Values preserved in description.`, ...item.activity],
  };
}
export function normalizeHierarchy(levels: Record<string, number>, defaultLevel: number) {
  const occupied = sortBy([...new Set([0, ...Object.values(levels)])]);
  return {
    levels: Object.fromEntries(Object.entries(levels).map(([id, level]) => [id, occupied.indexOf(level)])),
    defaultLevel: Math.max(0, occupied.indexOf(defaultLevel)),
  };
}
export function hierarchyConflicts(content: PreviewContent, levels: Record<string, number>) {
  return content.items.filter((child) => {
    const parent = content.items.find((item) => item.id === child.parent);
    return parent && levels[parent.type] !== levels[child.type] + 1;
  });
}
export function effectiveProperty(property: PreviewProperty, project: string, type: string): PreviewProperty {
  return resolvePropertyContext(property, project, type).effective;
}
export function defaultPropertyValues(content: PreviewContent, project: string, typeId: string) {
  const type = content.types.find((candidate) => candidate.id === typeId);
  const values: PreviewItem["values"] = {};
  for (const source of content.properties) {
    if (!type?.properties.includes(source.id)) continue;
    const property = effectiveProperty(source, project, typeId);
    if (!property.active || property.required || !property.defaultValue) continue;
    if (!["Number", "Dropdown", "Member picker"].includes(property.kind)) continue;
    values[property.id] = property.multiple ? [property.defaultValue] : property.defaultValue;
  }
  return values;
}
export function removedCascadeValues(content: PreviewContent, before: PreviewProperty, after: PreviewProperty) {
  return content.items.filter((item) => {
    const previous = effectiveProperty(before, item.project, item.type);
    const next = effectiveProperty(after, item.project, item.type);
    const values = item.values[before.id];
    return (
      Array.isArray(values) &&
      values.some(
        (id) => previous.options.some((option) => option.id === id) && !next.options.some((option) => option.id === id)
      )
    );
  });
}
const overlap = (left: string[], right: string[]) =>
  !left.length || !right.length || left.some((id) => right.includes(id));
export function contextOverlap(a: { projects: string[]; types: string[] }, b: { projects: string[]; types: string[] }) {
  return (
    Number(!!a.projects.length) + Number(!!a.types.length) === Number(!!b.projects.length) + Number(!!b.types.length) &&
    overlap(a.projects, b.projects) &&
    overlap(a.types, b.types)
  );
}
