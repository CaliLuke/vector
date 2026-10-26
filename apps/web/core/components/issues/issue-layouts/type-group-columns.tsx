import type { IGroupByColumn } from "@plane/types";
import type { WorkItemTypeIdentity } from "@/services/work-item-types.service";
import { TypeIcon } from "@/components/work-item-types/controls";
import { useWorkItemGroupingCatalog } from "@/hooks/use-work-item-grouping-catalog";

export function typeGroupColumns(types: WorkItemTypeIdentity[]): IGroupByColumn[] {
  return [
    ...types.map((type) => ({
      id: type.id,
      name: type.name,
      icon: <TypeIcon type={type} size="sm" />,
      payload: { type_id: type.id },
      isDropDisabled: true,
    })),
    { id: "None", name: "No type", payload: {}, isDropDisabled: true },
  ];
}

export function TypeGroupingStatus({ active }: { active: boolean }) {
  const { identities, isError, retry, supported, denied } = useWorkItemGroupingCatalog(active);
  if (!active) return null;
  if (isError)
    return (
      <div role="alert" className="p-6 text-13">
        Work item types could not be loaded.{" "}
        <button type="button" className="text-accent-primary" onClick={() => void retry()}>
          Retry
        </button>
      </div>
    );
  if (!supported || denied || identities?.enabled === false)
    return (
      <output className="block p-6 text-13">
        Work item type grouping is unavailable. Choose another group in Display.
      </output>
    );
  return <output className="block p-6 text-13">Loading work item types…</output>;
}
