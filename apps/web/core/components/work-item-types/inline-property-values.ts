import type { ItemSchema } from "@/services/work-item-schema.service";
import { editableProperty, submittedPropertyValues } from "./item-property-draft";
import type { PropertyValues } from "./item-property-draft";

/** Only explicit local edits follow an intentional reload; untouched values use
 * the new canonical source. Hidden or read-only edited fields require an explicit discard. */
export function reloadPropertyDraft(
  values: PropertyValues,
  previous: PropertyValues,
  schema: ItemSchema,
  current: PropertyValues,
  currentSchema: ItemSchema
) {
  const patch = submittedPropertyValues(values, schema, previous);
  const editable = new Set(
    currentSchema.enabled ? currentSchema.properties.filter(editableProperty).map((field) => field.id) : []
  );
  const unavailable = schema.properties.filter((field) => Object.hasOwn(patch, field.id) && !editable.has(field.id));
  // Compare canonical values to find actual changes, but retain local keystrokes
  // (such as a number ending in a decimal point) in the form after reloading.
  const changed = Object.keys(patch);
  return { values: { ...current, ...Object.fromEntries(changed.map((id) => [id, values[id]])) }, changed, unavailable };
}
