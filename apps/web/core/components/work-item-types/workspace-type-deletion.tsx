import { useState } from "react";
import type { PreviewType } from "./preview-content";
import type { TypePersistence } from "./library-persistence";
import { Confirmation } from "./controls";
import { TypeRemovalDialog } from "./type-removal-dialog";

/** Keep the chosen flow mounted when its successful write refreshes the catalog. */
export function WorkspaceTypeDeletion({
  type,
  persistence,
  revision,
  onClose,
}: {
  type: PreviewType;
  persistence: TypePersistence;
  revision: number;
  onClose: () => void;
}) {
  const [requiresReplacement] = useState(() => (persistence.types.find((item) => item.id === type.id)?.usage ?? 0) > 0);
  if (requiresReplacement)
    return (
      <TypeRemovalDialog
        workspaceSlug={persistence.propertyRemoval.scope}
        type={type}
        types={persistence.types}
        revision={revision}
        refreshRevision={persistence.propertyRemoval.refreshRevision}
        onClose={onClose}
      />
    );
  return (
    <Confirmation
      title={`Delete ${type.name}?`}
      action="Delete type"
      onClose={onClose}
      onConfirm={async () => {
        await persistence.remove(type.id, revision);
        onClose();
      }}
    >
      <p>This removes the unused type from the workspace and its projects.</p>
    </Confirmation>
  );
}
