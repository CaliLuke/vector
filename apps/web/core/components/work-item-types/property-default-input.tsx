import { previewPeople } from "./preview-content";
import type { PreviewProperty } from "./preview-content";
import { inputStyle } from "./controls";

type DefaultProperty = Pick<PreviewProperty, "kind" | "multiple" | "required" | "options">;
export function PropertyDefaultInput({
  id,
  property,
  members,
  value,
  onChange,
}: {
  id: string;
  property: DefaultProperty;
  members?: { id: string; name: string }[];
  value: string;
  onChange: (value: string) => void;
}) {
  if (property.kind === "Number")
    return (
      <input
        id={id}
        type="number"
        step="any"
        className={inputStyle}
        disabled={property.required}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    );
  const choices =
    property.kind === "Member picker"
      ? (members ?? previewPeople.map((name) => ({ id: name, name })))
      : property.options;
  return (
    <select
      id={id}
      multiple={property.multiple}
      disabled={property.required}
      className={inputStyle}
      value={property.multiple ? value.split(",").filter(Boolean) : value}
      onChange={(event) =>
        onChange(
          Array.from(event.target.selectedOptions)
            .map((option) => option.value)
            .filter(Boolean)
            .join(",")
        )
      }
    >
      <option value="">No default</option>
      {choices.map((option) => (
        <option key={option.id} value={option.id}>
          {option.name}
        </option>
      ))}
    </select>
  );
}

export function propertyDefaultLabel(property: PreviewProperty, members?: { id: string; name: string }[]) {
  if (!property.defaultValue) return "None";
  if (!["Dropdown", "Member picker"].includes(property.kind)) return property.defaultValue;
  const choices = property.kind === "Member picker" ? (members ?? []) : property.options;
  return property.defaultValue
    .split(",")
    .map((id) => choices.find((choice) => choice.id === id)?.name ?? "Unavailable selection")
    .join(", ");
}
