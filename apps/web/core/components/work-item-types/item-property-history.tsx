import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { History } from "lucide-react";
import { Button } from "@plane/propel/button";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { itemDetailSchemaOptions } from "@/queries/work-item-detail-schema";
import type { PropertyHistoryScope } from "@/queries/work-item-property-history";
import type { PropertyHistoryTarget, PropertyVersionDetail } from "@/services/work-item-property-history.service";
import { PreviewDialog } from "./controls";
import { HistoryLoadError, PropertyHistoryBrowser } from "./property-history-browser";
import { PropertyHistoryRestore } from "./property-history-restore";

export const ItemPropertyHistory = function ItemPropertyHistory({
  disabled = false,
  ...target
}: PropertyHistoryTarget & { disabled?: boolean }) {
  const client = useQueryClient();
  const accountId = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const permissions = useProjectPermissions(target.workspaceSlug, target.projectId);
  const item = useWorkItemDetail(target.workspaceSlug, target.projectId, target.issueId).issue;
  if (!permissions.can([5, 15, 20]) || permissions.isError || permissions.isPending) return null;
  const membership = permissions.membershipQuery.data;
  const scope = {
    ...target,
    accountId,
    generation,
    membershipRevision: membership?.updated_at,
    membershipRole: permissions.workspaceRole,
    projectRole: permissions.projectRole,
  };
  return (
    <HistoryEntry
      key={JSON.stringify([
        accountId,
        generation,
        target,
        scope.membershipRevision,
        scope.membershipRole,
        scope.projectRole,
      ])}
      scope={scope}
      editable={
        !disabled &&
        permissions.can([15, 20]) &&
        !!item &&
        !item.archived_at &&
        !item.is_draft &&
        item.project_id === target.projectId
      }
    />
  );
};
function HistoryEntry({ scope, editable }: { scope: PropertyHistoryScope; editable: boolean }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="my-3 inline-flex items-center gap-1.5 rounded px-1 py-1 text-12 text-secondary hover:bg-layer-2 focus-visible:outline-2 focus-visible:outline-accent-strong"
      >
        <History className="size-3.5" aria-hidden />
        Property history
      </button>
      {open && <HistoryDialog scope={scope} editable={editable} onClose={() => setOpen(false)} />}
    </>
  );
}
function HistoryDialog({
  scope,
  editable,
  onClose,
}: {
  scope: PropertyHistoryScope;
  editable: boolean;
  onClose: () => void;
}) {
  const [version, setVersion] = useState<PropertyVersionDetail>();
  const [busy, setBusy] = useState(false);
  const [blocked, setBlocked] = useState(false);
  const [message, setMessage] = useState<string>();
  const schema = useQuery(itemDetailSchemaOptions(scope));
  const restorable = schema.isSuccess
    ? schema.data.properties
        .filter((field) => field.kind === "Rich text" && field.configuration.active)
        .map((field) => field.id)
    : [];
  if (blocked)
    return (
      <PreviewDialog
        title="Property history"
        onClose={onClose}
        footer={
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
        }
      >
        <HistoryLoadError retry={() => setBlocked(false)}>
          This work item's history is no longer available. Check access and retry.
        </HistoryLoadError>
      </PreviewDialog>
    );
  return (
    <PreviewDialog
      title={version && editable ? "Restore property revision" : "Property history"}
      wide
      onClose={onClose}
      dismissDisabled={busy}
      footer={
        <Button variant="secondary" disabled={busy} onClick={onClose}>
          Close
        </Button>
      }
    >
      {message && <output className="mb-4 block text-13">{message}</output>}
      <div hidden={!!version && editable}>
        <PropertyHistoryBrowser
          scope={scope}
          editable={editable}
          restorable={restorable}
          onRestore={(chosen) => {
            setMessage(undefined);
            setVersion(chosen);
          }}
        />
      </div>
      {version && editable && (
        <PropertyHistoryRestore
          key={version.id}
          scope={scope}
          version={version}
          onBusy={setBusy}
          onAccessLost={() => {
            setVersion(undefined);
            setMessage(undefined);
            setBlocked(true);
          }}
          onBack={() => setVersion(undefined)}
          onSaved={(savedMessage, moved) => {
            if (moved) {
              onClose();
              return;
            }
            setVersion(undefined);
            setMessage(savedMessage);
          }}
        />
      )}
    </PreviewDialog>
  );
}
