import { useEffect, useEffectEvent } from "react";
import { Button } from "@plane/propel/button";
import { useItemEditLease } from "@/components/issues/use-item-edit-coordination";
import { ItemEditConfirmation, ItemEditRecovery } from "@/components/issues/issue-modal/item-edit-confirmation";
import type { PropertyHistoryScope } from "@/queries/work-item-property-history";
import type { PropertyVersionDetail } from "@/services/work-item-property-history.service";
import { usePropertyRestore } from "./use-property-restore";
import { HistoryLoadError, HistoryRichText, VersionAttribution } from "./property-history-browser";

type Props = {
  scope: PropertyHistoryScope;
  version: PropertyVersionDetail;
  onBack: () => void;
  onSaved: (message: string, moved: boolean) => void;
  onBusy: (busy: boolean) => void;
  onAccessLost: () => void;
};
export function PropertyHistoryRestore(props: Props) {
  const { held, blocked, acquire } = useItemEditLease(
    `${props.scope.workspaceSlug}/${props.scope.projectId}/${props.scope.issueId}`
  );
  useEffect(() => {
    if (!blocked) acquire();
  }, [blocked, acquire]);
  if (!held)
    return (
      <div className="space-y-4">
        <output className="block">
          Waiting for other edits to finish saving. Close history to retry any failed title or description saves.
        </output>
        <Button variant="secondary" onClick={props.onBack}>
          Back to history
        </Button>
      </div>
    );
  return <RestoreEditor {...props} />;
}
function RestoreEditor({ scope, version, onBack, onSaved, onBusy, onAccessLost }: Props) {
  const restore = usePropertyRestore(scope, version, onSaved, onAccessLost);
  const report = useEffectEvent(onBusy);
  useEffect(() => {
    report(restore.busy);
    return () => report(false);
  }, [restore.busy]);
  if (restore.edits.recoveryCommand)
    return (
      <ItemEditRecovery
        command={restore.edits.recoveryCommand}
        pending={restore.busy}
        onRetry={restore.recover}
        onClose={onBack}
      />
    );
  if (restore.source.isError)
    return (
      <HistoryLoadError retry={() => void restore.reload()}>Could not load the current work item.</HistoryLoadError>
    );
  if (!restore.source.isSuccess) return <output className="block">Loading the current work item…</output>;
  const field = restore.source.data.schema.properties.find(
    (candidate) =>
      candidate.id === version.property_id && candidate.kind === "Rich text" && candidate.configuration.active
  );
  if (!field)
    return (
      <div className="space-y-4">
        <p>This property is no longer active on the work item. Its earlier revisions remain available here.</p>
        <Button variant="secondary" onClick={onBack}>
          Back to history
        </Button>
      </div>
    );
  const currentValue = restore.source.data.schema.stored_values[version.property_id];
  const unchanged = (currentValue ?? null) === version.value;
  return (
    <div className="space-y-4">
      <p className="text-13">
        Restore <strong>{field.title}</strong> to this revision? The current value will remain in its history.
      </p>
      <div className="text-12">
        <VersionAttribution version={version} />
      </div>
      <HistoryRichText scope={scope} versionId={version.id} value={version.value} />
      {unchanged && <p className="text-13 text-secondary">This revision already matches the current value.</p>}
      {restore.error && (
        <p role="alert" className="text-13 text-danger-primary">
          {restore.error}
        </p>
      )}
      <div className="flex flex-wrap justify-end gap-2">
        <Button variant="secondary" disabled={restore.busy} onClick={onBack}>
          Back to history
        </Button>
        {restore.conflict ? (
          <Button
            variant="primary"
            disabled={restore.busy || restore.source.isFetching}
            onClick={() => void restore.reload()}
          >
            Reload current work item
          </Button>
        ) : (
          <Button
            variant="primary"
            disabled={unchanged || restore.busy}
            loading={restore.busy}
            onClick={() => void restore.restore().catch(() => {})}
          >
            Restore revision
          </Button>
        )}
      </div>
      {restore.edits.review && (
        <ItemEditConfirmation
          workspaceSlug={scope.workspaceSlug}
          review={restore.edits.review.value}
          command={restore.edits.review.command}
          onDecide={restore.edits.decide}
        />
      )}
    </div>
  );
}
