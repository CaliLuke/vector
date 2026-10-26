import { useTablePropertyValue } from "@/hooks/use-work-item-table-values";
import type { TIssue, TWorkItemProjectedValue } from "@plane/types";
import type { WorkItemPropertyIdentity } from "@/services/work-item-types.service";
import { PropertyCellValue, projectedCellValue } from "./property-cell-value";

type Props = { issue: TIssue; field: WorkItemPropertyIdentity; onEdit: (issue: TIssue) => void; disabled: boolean };
export function PropertyColumn({ issue, field, onEdit, disabled }: Props) {
  const result = useTablePropertyValue(issue.id);
  const values = result?.status === "ready" ? result.data?.property_values : undefined;
  const projection = cellProjection(field, values);
  const canEdit =
    result?.status === "ready" &&
    !disabled &&
    field.kind !== "Formula" &&
    projectedCellValue(field, projection).status !== "not_applicable";
  return (
    <td tabIndex={0} className="h-11 max-w-72 min-w-44 border-r border-b border-subtle px-3 text-13">
      {result?.status === "error" || result?.status === "unavailable" ? (
        <span className="text-tertiary">Unavailable</span>
      ) : canEdit && field.kind !== "URL" ? (
        <button
          type="button"
          className="clickable block w-full truncate text-left focus-visible:outline-2 focus-visible:outline-accent-strong"
          aria-label={`Edit ${field.name} for ${issue.name}`}
          onClick={() => onEdit(issue)}
        >
          <PropertyCellValue field={field} projection={projection} />
        </button>
      ) : (
        <div className="truncate">
          <PropertyCellValue field={field} projection={projection} />
        </div>
      )}
    </td>
  );
}

function cellProjection(
  field: WorkItemPropertyIdentity,
  values: TIssue["property_values"]
): TWorkItemProjectedValue | undefined {
  if (!values) return undefined;
  return (
    values[field.property_id] ?? {
      kind: field.kind,
      context_id: "",
      catalog_id: field.catalog_id,
      status: "not_applicable",
      value: null,
      error: "",
    }
  );
}
