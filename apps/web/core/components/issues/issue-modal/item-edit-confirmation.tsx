import { useState } from "react";
import { Button } from "@plane/propel/button";
import { RichTextEditor } from "@/components/editor/rich-text";
import { PreviewDialog } from "@/components/work-item-types/controls";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import type { ItemEditCommand, ItemEditReview } from "@/services/work-item-edits.service";

export function ItemEditConfirmation({
  workspaceSlug,
  review,
  command,
  onDecide,
}: {
  workspaceSlug: string;
  review: ItemEditReview;
  command: ItemEditCommand;
  onDecide: (accepted: boolean) => void;
}) {
  const workspace = useWorkspaceBySlug(workspaceSlug);
  return (
    <PreviewDialog
      title="Keep property values in the description"
      description={`${review.preserved_properties} ${review.preserved_properties === 1 ? "property no longer applies" : "properties no longer apply"}. Their values will be added to the description below. Switching back will not restore these values to their old fields.`}
      wide
      onClose={() => onDecide(false)}
      footer={
        <>
          <Button variant="secondary" onClick={() => onDecide(false)}>
            Cancel
          </Button>
          <Button variant="primary" disabled={!workspace.data} onClick={() => onDecide(true)}>
            {command.mode === "publish" ? "Publish work item" : "Save changes"}
          </Button>
        </>
      }
    >
      {workspace.data ? (
        <RichTextEditor
          id={`item-edit-review-${review.id}`}
          initialValue={review.description_html}
          workspaceSlug={workspaceSlug}
          workspaceId={workspace.data.id}
          projectId={command.project_id || undefined}
          editable={false}
          containerClassName="border-none"
        />
      ) : (
        <output>
          {workspace.isError ? (
            <>
              Could not load the workspace.{" "}
              <button type="button" onClick={() => void workspace.refetch()}>
                Retry
              </button>
            </>
          ) : (
            "Loading description…"
          )}
        </output>
      )}
    </PreviewDialog>
  );
}

export function ItemEditRecovery({
  command,
  pending,
  onRetry,
  onClose,
}: {
  command: ItemEditCommand;
  pending: boolean;
  onRetry: (command: ItemEditCommand) => Promise<void>;
  onClose: () => void;
}) {
  const [error, setError] = useState(false);
  return (
    <div className="rounded-lg bg-surface-1 p-5">
      <h3 className="text-h4-medium">Confirm saved work item</h3>
      <p className="mt-3 text-13 text-secondary">
        {pending ? "Confirming the save…" : "Confirm the result of the previous save before making more changes."}
      </p>
      {command.changes.name && <p className="mt-2 text-13">{command.changes.name}</p>}
      {error && (
        <p role="alert" className="mt-3 text-13 text-danger-primary">
          Could not confirm the save. Your original request is retained; try again.
        </p>
      )}
      <div className="mt-5 flex justify-end gap-3">
        <Button variant="secondary" onClick={onClose} disabled={pending}>
          Close
        </Button>
        <Button
          variant="primary"
          loading={pending}
          disabled={pending}
          onClick={() => {
            setError(false);
            void onRetry(command).catch(() => setError(true));
          }}
        >
          Confirm save
        </Button>
      </div>
    </div>
  );
}
