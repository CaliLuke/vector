import type { TWorkItemProjectedValue } from "@plane/types";
import { sanitizeHTML } from "@plane/utils/string";
import type { WorkItemPropertyIdentity } from "@/services/work-item-types.service";
import { dateLabel, scalarLabel } from "@/components/work-item-types/property-value-label";
import { dropdownPath, OptionIcon } from "@/components/work-item-types/property-option-label";

/** Cascade columns have identities per catalogue/level, never an ordinal path slot. */
export function projectedCellValue(field: WorkItemPropertyIdentity, projection?: TWorkItemProjectedValue) {
  if (!projection) return { status: "loading" as const, value: null };
  if (field.kind === "Cascading" && field.catalog_id !== projection.catalog_id)
    return { status: "not_applicable" as const, value: null };
  if (projection.status !== "value" || field.kind !== "Cascading") return projection;
  const value = projection.levels?.[field.level_id] ?? null;
  return { ...projection, status: value === null ? ("empty" as const) : ("value" as const), value };
}

export function PropertyCellValue({
  field,
  projection,
}: {
  field: WorkItemPropertyIdentity;
  projection?: TWorkItemProjectedValue;
}) {
  const result = projectedCellValue(field, projection);
  if (result.status === "loading") return <span className="text-tertiary">Loading…</span>;
  if (result.status === "not_applicable")
    return (
      <span className="text-placeholder" title="Not applicable">
        —
      </span>
    );
  if (result.status === "empty")
    return (
      <span className="text-placeholder" title="No value">
        —
      </span>
    );
  if (result.status === "error")
    return (
      <span className="text-danger-primary" title={result.error}>
        Calculation error
      </span>
    );
  const value = result.value;
  if (field.kind === "Rich text" && typeof value === "string")
    return <span className="block truncate" dangerouslySetInnerHTML={{ __html: sanitizeHTML(value) }} />;
  if (["Dropdown", "Cascading", "Member picker", "Release picker"].includes(field.kind))
    return <SelectionCell field={field} value={value} />;
  if ((field.kind === "Date" || field.result_kind === "Date") && typeof value === "string")
    return <time dateTime={value}>{dateLabel(value, projection?.format)}</time>;
  if (field.kind === "URL" && typeof value === "string" && /^https?:\/\//i.test(value))
    return (
      <a href={value} target="_blank" rel="noopener noreferrer" className="text-accent-primary underline">
        {value}
      </a>
    );
  return <span>{scalarLabel(value)}</span>;
}

function SelectionCell({ field, value }: { field: WorkItemPropertyIdentity; value: TWorkItemProjectedValue["value"] }) {
  const ids = Array.isArray(value) ? value : typeof value === "string" ? [value] : [];
  return (
    <span>
      {ids.map((id, index) => {
        const option = field.options.find((entry) => entry.id === id);
        const name = field.kind === "Dropdown" ? dropdownPath(id, field.options) : option?.name;
        return (
          <span key={id}>
            {index > 0 && ", "}
            {option?.icon && <OptionIcon icon={option.icon} />}
            {name || "Unavailable selection"}
          </span>
        );
      })}
    </span>
  );
}
