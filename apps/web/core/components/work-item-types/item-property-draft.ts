import type { ItemSchema, ResolvedProperty } from "@/services/work-item-schema.service";

export type PropertyValues = Record<string, unknown>;

export function editableProperty(property: ResolvedProperty) {
  return property.kind !== "Formula" && !(property.kind === "Text" && property.configuration.format === "Read-only");
}

/** Defaults are materialized only for a new item/draft and only for absent keys. */
export function applyPropertyDefaults(values: PropertyValues, schema: ItemSchema, creating: boolean): PropertyValues {
  if (!creating) return values;
  let next = values;
  for (const property of schema.properties) {
    const config = property.configuration;
    if (!editableProperty(property) || property.id in values || config.required || config.default_value == null)
      continue;
    if (next === values) next = { ...values };
    next[property.id] = Array.isArray(config.default_value) ? [...config.default_value] : config.default_value;
  }
  return next;
}

/** Hidden/inactive and other-type drafts remain local; they are not submitted as edits. */
export function submittedPropertyValues(
  values: PropertyValues,
  schema: ItemSchema,
  saved: PropertyValues = {}
): PropertyValues {
  return Object.fromEntries(
    schema.properties
      .filter((property) => editableProperty(property) && Object.hasOwn(values, property.id))
      .map((property) => [property.id, canonicalPropertyInput(property, values[property.id])] as const)
      .filter(([id, value]) => !Object.hasOwn(saved, id) || JSON.stringify(value) !== JSON.stringify(saved[id]))
  );
}

/** Former fields are validated under their source configuration by the reviewed
 * save endpoint. They are sent only when edited, so hidden stored values stay put. */
export function submittedFormerPropertyValues(
  values: PropertyValues,
  source: ItemSchema,
  destination: ItemSchema,
  saved: PropertyValues
): PropertyValues {
  const remaining = new Set(destination.properties.map((field) => field.id));
  return Object.fromEntries(
    Object.entries(submittedPropertyValues(values, source, saved)).filter(([id]) => !remaining.has(id))
  );
}

export function propertyFieldErrors(cause: unknown): Record<string, string> {
  if (!cause || typeof cause !== "object") return {};
  const error = cause as { response?: { data?: unknown }; data?: unknown };
  const body = error.response?.data ?? error.data ?? cause;
  const data = body && typeof body === "object" && "fields" in body ? body.fields : body;
  if (!data || typeof data !== "object" || !("custom_properties" in data)) return {};
  const fields = data.custom_properties;
  if (!fields || typeof fields !== "object" || Array.isArray(fields)) return {};
  return Object.fromEntries(
    Object.entries(fields).flatMap(([id, value]) => (typeof value === "string" ? [[id, value]] : []))
  );
}

/** Keep numeric keystrokes (including a trailing decimal point) in Form until submit/preview. */
function canonicalPropertyInput(property: ResolvedProperty, value: unknown): unknown {
  if (property.kind !== "Number" || typeof value !== "string") return value;
  if (!value.trim()) return null;
  if (!/^[+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?$/.test(value.trim())) return value;
  const number = Number(value);
  return Number.isFinite(number) ? number : value;
}
