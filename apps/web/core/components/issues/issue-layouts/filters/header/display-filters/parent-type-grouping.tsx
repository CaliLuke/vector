import type { IIssueDisplayFilterOptions, TIssueGroupByOptions } from "@plane/types";
import { CustomSelect } from "@plane/ui";
import { useWorkItemGroupingCatalog } from "@/hooks/use-work-item-grouping-catalog";
import { isParentTypeGrouping } from "@/helpers/issue-list/grouping-axis";
import { TypeIcon } from "@/components/work-item-types/controls";

export function ParentTypeGrouping({
  displayFilters,
  isEpic,
  onChange,
}: {
  displayFilters?: IIssueDisplayFilterOptions;
  isEpic?: boolean;
  onChange: (value: TIssueGroupByOptions) => void;
}) {
  const enabled = !isEpic && (displayFilters?.layout === "list" || displayFilters?.layout === "kanban");
  const catalog = useWorkItemGroupingCatalog(enabled);
  if (!enabled || !catalog.supported || !catalog.identities?.enabled) return null;
  const selected = isParentTypeGrouping(displayFilters?.group_by) ? displayFilters.group_by : "";
  const type = catalog.identities.types.find((item) => `parent_type_${item.id}` === selected);
  return (
    <div className="px-2 py-1">
      <CustomSelect
        value={selected}
        label={type ? `Parent type: ${type.name}` : "Parent type"}
        onChange={(value: `parent_type_${string}`) => onChange(value)}
        placement="bottom-start"
        buttonClassName="w-full text-13"
        optionsClassName="max-h-64 overflow-y-auto"
      >
        {catalog.identities.types.map((item) => (
          <CustomSelect.Option key={item.id} value={`parent_type_${item.id}`}>
            <span className="flex items-center gap-2">
              <TypeIcon type={item} size="sm" />
              {item.name}
            </span>
          </CustomSelect.Option>
        ))}
      </CustomSelect>
      {selected && !type && (
        <p role="alert" className="text-12 text-danger-primary">
          This parent type is unavailable. Choose another group.
        </p>
      )}
    </div>
  );
}
