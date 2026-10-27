import type { IIssueDisplayFilterOptions, TIssueGroupByOptions } from "@plane/types";
import { useWorkItemTableCatalog } from "@/hooks/use-work-item-table-catalog";
import { isCustomGrouping } from "@/helpers/issue-list/grouping-axis";
import { FilterOption } from "@/components/issues/issue-layouts/filters";

export function CustomPropertyGrouping({
  displayFilters,
  axis,
  isEpic,
  onChange,
}: {
  displayFilters?: IIssueDisplayFilterOptions;
  axis: "group_by" | "sub_group_by";
  isEpic?: boolean;
  onChange: (value: TIssueGroupByOptions) => void;
}) {
  const enabled = !isEpic && (displayFilters?.layout === "list" || displayFilters?.layout === "kanban");
  const catalog = useWorkItemTableCatalog(enabled);
  if (!enabled || !catalog.supported) return null;
  if (catalog.isError)
    return (
      <p role="alert" className="text-12 text-danger-primary">
        Could not load custom groups.{" "}
        <button type="button" onClick={() => void catalog.retry()}>
          Retry
        </button>
      </p>
    );
  const selected = displayFilters?.[axis];
  const other =
    displayFilters?.layout === "kanban" ? displayFilters[axis === "group_by" ? "sub_group_by" : "group_by"] : undefined;
  const fields = catalog.fields.filter((field) => field.groupable && field.key !== other);
  return (
    <>
      {fields.map((field) => (
        <FilterOption
          key={field.key}
          title={field.name}
          isChecked={selected === field.key}
          multiple={false}
          onClick={() => onChange(field.key)}
        />
      ))}
      {catalog.identities && isCustomGrouping(selected) && !fields.some((field) => field.key === selected) && (
        <p role="alert" className="text-12 text-danger-primary">
          This custom grouping is unavailable. Choose another group.
        </p>
      )}
    </>
  );
}
