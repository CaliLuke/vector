import { useEffect, useLayoutEffect, useState } from "react";
import { Button } from "@plane/propel/button";
import { useItemEditLease } from "@/components/issues/use-item-edit-coordination";
import { ItemEditConfirmation, ItemEditRecovery } from "@/components/issues/issue-modal/item-edit-confirmation";
import type { PublishedItemEditScope } from "@/queries/published-item-edit";
import { usePublishedItemEdit } from "./use-published-item-edit";
import { InlinePropertyForm } from "./inline-property-form";

type Props = {
  onCloseReadyChange?: (ready: boolean) => void;
  scope: PublishedItemEditScope;
  onClose: () => void;
  onSaved: () => void;
  onAccessLost: () => void;
};
export function InlinePropertyEditor(props: Props) {
  const { held, blocked, acquire } = useItemEditLease(
    `${props.scope.workspaceSlug}/${props.scope.projectId}/${props.scope.issueId}`
  );
  useEffect(() => {
    if (!blocked) acquire();
  }, [blocked, acquire]);
  if (!held)
    return (
      <div className="space-y-3">
        <output>Waiting for other edits to finish saving.</output>
        <Button variant="secondary" onClick={props.onClose}>
          Cancel
        </Button>
      </div>
    );
  return <InlinePropertySession {...props} />;
}
function InlinePropertySession({ scope, onClose, onSaved, onAccessLost, onCloseReadyChange }: Props) {
  const edit = usePublishedItemEdit(scope, onSaved, onAccessLost);
  const [formCloseReady, setFormCloseReady] = useState(true);
  useLayoutEffect(() => {
    onCloseReadyChange?.(!edit.busy && formCloseReady);
  }, [edit.busy, formCloseReady, onCloseReadyChange]);
  if (edit.edits.recoveryCommand)
    return (
      <ItemEditRecovery
        command={edit.edits.recoveryCommand}
        pending={edit.busy}
        onRetry={edit.recover}
        onClose={onClose}
      />
    );
  if (!edit.source.data && edit.source.isError)
    return (
      <div className="space-y-3">
        <p role="alert">Could not load the current work item.</p>
        <Button variant="secondary" onClick={() => void edit.reload()}>
          Retry
        </Button>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
      </div>
    );
  if (!edit.source.data)
    return (
      <div>
        <output>Loading current values…</output>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
      </div>
    );
  return (
    <>
      <InlinePropertyForm
        scope={scope}
        onCloseReadyChange={setFormCloseReady}
        edit={edit}
        snapshot={edit.source.data}
        onClose={onClose}
        onAccessLost={onAccessLost}
      />
      {edit.edits.review && (
        <ItemEditConfirmation
          workspaceSlug={scope.workspaceSlug}
          review={edit.edits.review.value}
          command={edit.edits.review.command}
          onDecide={edit.edits.decide}
        />
      )}
    </>
  );
}
