import type {
  PropertyConfiguration,
  PropertyDefinition,
  PropertyOption,
} from "@/services/work-item-properties.service";
import type { PreviewOption, PreviewProperty } from "./preview-content";

const textDefault = (value: PropertyConfiguration["default_value"]) =>
  Array.isArray(value) ? value.join(",") : value == null ? "" : String(value);
const previewOption = (value: PropertyOption): PreviewOption => ({ ...value, nextLabel: value.next_label });
const savedOption = (value: PreviewOption): PropertyOption => ({
  ...value,
  icon: value.icon ?? "",
  next_label: value.nextLabel,
});
export function propertyForEditor(def: PropertyDefinition, used: boolean): PreviewProperty {
  const config = def.configuration;
  return {
    id: def.id,
    internalName: def.name,
    icon: def.icon,
    name: def.title,
    description: def.description,
    kind: def.kind,
    active: config.active,
    required: config.required,
    multiple: config.multiple,
    format: config.format,
    defaultValue:
      def.kind === "Text" && config.format === "Read-only" ? config.content : textDefault(config.default_value),
    options: (config.options ?? []).map(previewOption),
    levels: config.levels ?? [],
    expression: config.expression,
    result: config.result,
    contexts: (def.contexts ?? []).map((context) => ({
      id: context.id,
      name: context.name,
      projects: context.project_ids ?? [],
      types: context.type_ids ?? [],
      active: context.override_keys.includes("active") ? context.configuration.active : null,
      required: context.override_keys.includes("required") ? context.configuration.required : null,
      defaultValue: context.override_keys.includes("default_value")
        ? textDefault(context.configuration.default_value)
        : null,
      format: context.override_keys.includes("format") ? context.configuration.format : null,
      options: context.override_keys.includes("options")
        ? (context.configuration.options ?? []).map(previewOption)
        : undefined,
    })),
    used,
  };
}
function defaultValue(value: PreviewProperty, text: string): PropertyConfiguration["default_value"] {
  if (!text.trim() || value.required) return null;
  if (value.kind === "Number") {
    const number = Number(text);
    if (!Number.isFinite(number)) throw new Error("Enter a finite numeric default value.");
    return number;
  }
  if (value.kind === "Dropdown" || value.kind === "Member picker")
    return value.multiple ? text.split(",").filter(Boolean) : text;
  return null;
}
export function propertyToSave(value: PreviewProperty): PropertyDefinition {
  const configuration: PropertyConfiguration = {
    active: value.active,
    required: value.required,
    multiple: value.multiple,
    format: value.kind === "Text" || value.kind === "Date" ? value.format : "",
    content: value.kind === "Text" && value.format === "Read-only" ? value.defaultValue : "",
    default_value: defaultValue(value, value.defaultValue),
    options: value.kind === "Dropdown" || value.kind === "Cascading" ? value.options.map(savedOption) : [],
    levels: value.kind === "Cascading" ? value.levels : [],
    expression: value.kind === "Formula" ? value.expression : "",
    result: value.kind === "Formula" ? value.result : "",
  };
  return {
    id: value.id,
    name: value.internalName ?? "",
    title: value.name,
    description: value.description,
    icon: value.icon ?? "",
    kind: value.kind,
    configuration,
    contexts: value.contexts.map((context) => {
      const override_keys: string[] = [];
      const config = { ...configuration };
      if (context.active != null) {
        override_keys.push("active");
        config.active = context.active;
      }
      if (context.required != null) {
        override_keys.push("required");
        config.required = context.required;
      }
      if (context.defaultValue != null) {
        override_keys.push("default_value");
        config.default_value = defaultValue({ ...value, required: config.required }, context.defaultValue);
      }
      if (context.format != null) {
        override_keys.push("format");
        config.format = context.format;
      }
      if (context.options) {
        override_keys.push("options");
        config.options = context.options.map(savedOption);
      }
      return {
        id: context.id,
        name: context.name,
        project_ids: context.projects,
        type_ids: context.types,
        override_keys,
        configuration: config,
      };
    }),
  };
}
