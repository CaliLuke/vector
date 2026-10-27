import type { IGroupByColumn } from "@plane/types";
import type { WorkItemPropertyIdentity } from "@/services/work-item-types.service";
import { OptionIcon } from "@/components/work-item-types/property-option-label";
import { useWorkItemTableCatalog } from "@/hooks/use-work-item-table-catalog";
import { isParentTypeGrouping, isCustomGrouping } from "@/helpers/issue-list/grouping-axis";
import type { IssueListGroupedIds } from "@/queries/issue-lists";
import { TypeGroupingStatus } from "./type-group-columns";

export function propertyGroupingIds(groups: IssueListGroupedIds | undefined, subgroup = false) {
  if (!groups) return [];
  return subgroup
    ? [...new Set(Object.values(groups).flatMap((value) => (Array.isArray(value) ? [] : Object.keys(value))))]
    : Object.keys(groups);
}

export function GroupingStatus({ groupBy, subGroupBy }: { groupBy: unknown; subGroupBy?: unknown }) {
  if (isParentTypeGrouping(groupBy)) return <output className="block p-6 text-13">Loading parent groups…</output>;
  return isCustomGrouping(groupBy) || isCustomGrouping(subGroupBy) ? (
    <PropertyGroupingStatus />
  ) : (
    <TypeGroupingStatus active={groupBy === "type_id" || subGroupBy === "type_id"} />
  );
}

export function propertyGroupColumns(field: WorkItemPropertyIdentity, observed: string[]): IGroupByColumn[] {
  const options =
    field.kind === "Boolean"
      ? [
          { id: "true", name: "True" },
          { id: "false", name: "False" },
        ]
      : field.options;
  const labels = new Map(options.map((option) => [option.id, option]));
  const reserved = new Map([
    ["None", "No value"],
    ["NotApplicable", "Not applicable"],
    ["Error", "Invalid value"],
  ]);
  return [...new Set([...options.map((option) => option.id), ...observed, ...reserved.keys()])].map((id) => ({
    id,
    name: labels.get(id)?.name ?? reserved.get(id) ?? "Unavailable value",
    icon: <OptionIcon icon={field.icon} />,
    payload: {},
    isDropDisabled: true,
  }));
}

function PropertyGroupingStatus() {
  const catalog = useWorkItemTableCatalog();
  if (catalog.isError)
    return (
      <div role="alert" className="p-6 text-13">
        Custom groups could not be loaded.{" "}
        <button type="button" onClick={() => void catalog.retry()}>
          Retry
        </button>
      </div>
    );
  return (
    <output className="block p-6 text-13">
      {catalog.identities
        ? "This custom grouping is unavailable. Choose another group in Display."
        : "Loading custom groups…"}
    </output>
  );
}
