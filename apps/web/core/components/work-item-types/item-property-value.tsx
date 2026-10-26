import { scalarLabel, dateLabel } from "./property-value-label";
import type { ReactNode } from "react";
import type { ItemSchema, ResolvedProperty } from "@/services/work-item-schema.service";
import { OptionIcon, dropdownPath } from "./property-option-label";

type Props = {
  property: ResolvedProperty;
  schema: ItemSchema;
  renderRichText: (html: string) => ReactNode;
};

export function ItemPropertyValue({ property, schema, renderRichText }: Props) {
  const value = schema.stored_values[property.id];
  const config = property.configuration;
  if (property.kind === "Formula") return <FormulaValue result={schema.computed[property.id]} />;
  if (property.kind === "Text" && config.format === "Read-only")
    return <span className="whitespace-pre-wrap">{config.content}</span>;
  if (emptyValue(value)) return <span className="text-placeholder">—</span>;
  if (property.kind === "Rich text") return renderRichText(typeof value === "string" ? value : "");
  if (["Dropdown", "Cascading", "Member picker", "Release picker"].includes(property.kind))
    return <SelectionValue property={property} schema={schema} value={value} />;
  if (property.kind === "Date" && typeof value === "string")
    return <time dateTime={value}>{dateLabel(value, config.format)}</time>;
  if (property.kind === "URL" && typeof value === "string")
    return /^https?:\/\//i.test(value) ? (
      <a className="text-accent-primary underline" href={value} target="_blank" rel="noopener noreferrer">
        {value}
      </a>
    ) : (
      <span>{value}</span>
    );
  return <span className="whitespace-pre-wrap">{scalarLabel(value)}</span>;
}

function SelectionValue({ property, schema, value }: Omit<Props, "renderRichText"> & { value: unknown }) {
  const ids = Array.isArray(value)
    ? value.filter((id): id is string => typeof id === "string")
    : typeof value === "string"
      ? [value]
      : [];
  const references = property.kind === "Member picker" ? schema.members : schema.releases;
  return (
    <span>
      {ids.map((id, index) => {
        const option = property.configuration.options.find((entry) => entry.id === id);
        const name =
          property.kind === "Dropdown"
            ? dropdownPath(id, property.configuration.options)
            : property.kind === "Cascading"
              ? option?.name
              : references.find((entry) => entry.id === id)?.name;
        return (
          <span key={id}>
            {index > 0 && (property.kind === "Cascading" ? " → " : ", ")}
            {option?.icon && <OptionIcon icon={option.icon} />}
            {name || "Unavailable selection"}
          </span>
        );
      })}
    </span>
  );
}

function emptyValue(value: unknown) {
  return value === null || value === undefined || value === "" || (Array.isArray(value) && value.length === 0);
}
function FormulaValue({ result }: { result: ItemSchema["computed"][string] | undefined }) {
  const label = !result
    ? "Calculating…"
    : result.status === "error"
      ? result.error
      : result.status === "empty"
        ? "—"
        : scalarLabel(result.value);
  return <output className={result?.status === "error" ? "text-danger-primary" : undefined}>{label}</output>;
}
