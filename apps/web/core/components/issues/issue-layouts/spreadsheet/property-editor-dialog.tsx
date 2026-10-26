import { useState } from "react";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { useQueryClient } from "@tanstack/react-query";
import { PreviewDialog } from "@/components/work-item-types/controls";
import { InlinePropertyEditor } from "@/components/work-item-types/inline-property-editor";
import { useWorkItemCatalogAccess } from "@/hooks/use-work-item-catalog-access";
import type { PublishedItemEditScope } from "@/queries/published-item-edit";
import { forgetItemReads } from "@/queries/work-item-access";

export function PropertyEditorDialog({
  workspaceSlug,
  projectId,
  issueId,
  onClose,
}: {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  onClose: () => void;
}) {
  const client = useQueryClient();
  const [canClose, setCanClose] = useState(true);
  const access = useWorkItemCatalogAccess(workspaceSlug, projectId, true);
  const [opened, setOpened] = useState<{
    scope: PublishedItemEditScope;
    access: ReturnType<typeof captureProjectAccessScope>;
  }>();
  if (!opened && access.allowed && access.permissions.can([15, 20])) {
    const scope = { ...access.scope, workspaceSlug, projectId, issueId };
    setOpened({ scope, access: captureProjectAccessScope(client, scope) });
  }
  // Preserve the opened draft across transient verification failures. The editor
  // requires verified access before reads/writes, and hard revocation still closes it.
  const permitted = opened?.access.isCurrent();
  return (
    <PreviewDialog
      title="Edit custom properties"
      onClose={onClose}
      footer={null}
      dismissDisabled={!!permitted && !canClose}
      wide
    >
      {permitted && opened ? (
        <InlinePropertyEditor
          key={JSON.stringify(opened.scope)}
          scope={opened.scope}
          onCloseReadyChange={setCanClose}
          onClose={onClose}
          onSaved={onClose}
          onAccessLost={() => {
            forgetItemReads(client, opened.scope);
            onClose();
          }}
        />
      ) : (
        <p role="alert">
          {access.permissionError
            ? "Could not verify work item access."
            : access.permissions.isPending
              ? "Checking work item access…"
              : "You no longer have permission to edit this work item."}
        </p>
      )}
    </PreviewDialog>
  );
}
