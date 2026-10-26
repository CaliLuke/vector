import { useWorkItemTypeCatalog } from "@/hooks/use-work-item-type-catalog";
import { TypeIcon } from "./controls";

export function WorkItemTypeBadge({
  workspaceSlug,
  projectId,
  typeId,
}: {
  workspaceSlug: string;
  projectId: string;
  typeId: string | null | undefined;
}) {
  const { catalog, permissions } = useWorkItemTypeCatalog(workspaceSlug, projectId);
  if (permissions.isError || (!permissions.isPending && permissions.projectRole === undefined)) return null;
  if (permissions.isPending || catalog.isPending)
    return <output aria-label="Loading work item type" className="size-5 animate-pulse rounded bg-layer-2" />;
  if (catalog.isError) return <span className="text-11 text-tertiary">Type unavailable</span>;
  if (!catalog.data.enabled) return null;
  const type = catalog.data.types.find((entry) => entry.id === typeId);
  if (!type) return <span className="text-11 text-tertiary">{typeId === null ? "No type" : "Unavailable type"}</span>;
  return (
    <span className="inline-flex min-w-0 items-center gap-1 text-11 text-secondary" title={type.name}>
      <TypeIcon type={type} size="sm" />
      <span className="max-w-32 truncate">{type.name}</span>
    </span>
  );
}
