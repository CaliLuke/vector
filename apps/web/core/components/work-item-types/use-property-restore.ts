import type { PropertyHistoryScope } from "@/queries/work-item-property-history";
import type { PropertyVersionDetail } from "@/services/work-item-property-history.service";
import { usePublishedItemEdit } from "./use-published-item-edit";

export function usePropertyRestore(
  scope: PropertyHistoryScope,
  version: PropertyVersionDetail,
  onSaved: (message: string, moved: boolean) => void,
  onAccessLost: () => void
) {
  const edit = usePublishedItemEdit(
    scope,
    (accepted, moved) =>
      onSaved(
        accepted.restore?.property_id === version.property_id && accepted.restore.version_id === version.id
          ? "Revision restored."
          : "Previous save confirmed.",
        moved
      ),
    onAccessLost,
    true
  );
  const restore = async () => {
    if (!edit.source.data || edit.conflict) return;
    await edit.save({
      mode: "edit",
      item_id: scope.issueId,
      project_id: scope.projectId,
      revision: edit.source.data.schema.revision,
      expected_source_version: edit.source.data.source.source_version,
      changes: {},
      restore: { property_id: version.property_id, version_id: version.id },
    });
  };
  return { ...edit, restore };
}
