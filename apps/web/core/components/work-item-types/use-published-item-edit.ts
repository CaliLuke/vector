import { isItemAccessDenied, forgetItemReads } from "@/queries/work-item-access";
import { useItemEditReconciliation } from "./use-item-edit-reconciliation";
import { useEffect, useEffectEvent, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useReviewedItemEdit } from "@/components/issues/issue-modal/use-reviewed-item-edit";
import {
  issueSourceChanged,
  issueConfigurationChanged,
  issueSaveError,
} from "@/components/issues/issue-modal/issue-field-errors";
import { captureItemLineageScope, captureItemReadScope } from "@/queries/work-item-read-scope";
import { getIdentitySessionGeneration } from "@/queries/identity";
import { isPropertyHistoryAccessDenied } from "@/queries/work-item-property-history";
import { publishedItemEditSourceOptions } from "@/queries/published-item-edit";
import type { PublishedItemEditScope } from "@/queries/published-item-edit";
import { workItemEditsService } from "@/services/work-item-edits.service";
import type { ItemEditCommand } from "@/services/work-item-edits.service";

/** Mounted only while the containing panel holds the shared item editor lease. */
export function usePublishedItemEdit(
  scope: PublishedItemEditScope,
  onSaved: (accepted: ItemEditCommand, moved: boolean) => void,
  onAccessLost: () => void,
  reloadAfterValidation = false
) {
  const client = useQueryClient();
  const [intentId] = useState(() => crypto.randomUUID());
  const [error, setError] = useState<string>();
  const [cause, setCause] = useState<unknown>();
  const [conflict, setConflict] = useState(false);
  const [reconciling, setReconciling] = useState(false);
  const options = publishedItemEditSourceOptions(scope, intentId);
  const captureAccess = (captureCurrentScope: typeof captureItemReadScope) => {
    const current = captureCurrentScope(client, scope, options.queryKey);
    return () => {
      const currentScope = current();
      const denied = isItemAccessDenied(client, scope);
      const generationCurrent = getIdentitySessionGeneration(client) === scope.generation;
      if (!currentScope || denied || !generationCurrent) throw new Error("Work item access changed.");
    };
  };
  const captureReadAccess = () => captureAccess(captureItemReadScope);
  const captureReceiptAccess = () => captureAccess(captureItemLineageScope);
  const edits = useReviewedItemEdit(scope.workspaceSlug, scope.issueId, {
    captureAccess: captureReadAccess,
    preview: (command, beforeSend) => workItemEditsService.preview(scope.workspaceSlug, command, beforeSend),
    commit: (command, beforeSend) => workItemEditsService.commit(scope.workspaceSlug, command, beforeSend),
  });
  const source = useQuery({ ...options, enabled: !edits.recoveryCommand });
  const blockHistory = useEffectEvent(() => {
    forgetItemReads(client, scope);
    onAccessLost();
  });
  useEffect(() => {
    if (isPropertyHistoryAccessDenied(source.error)) blockHistory();
  }, [source.error]);
  const reconciliation = useItemEditReconciliation(scope, options.queryKey);
  const save = async (command: ItemEditCommand) => {
    if (edits.isPending || reconciling) return;
    const access = captureReceiptAccess();
    const current = () => {
      try {
        access();
        return edits.isCurrent();
      } catch {
        return false;
      }
    };
    captureReadAccess()();
    setError(undefined);
    setCause(undefined);
    setReconciling(true);
    try {
      const listOwners = await reconciliation.captureListOwners(command.project_id ?? scope.projectId);
      const { saved, command: accepted } = await edits.submitWithReceipt(command);
      if (!current()) return;
      const moved = await reconciliation.reconcile(saved, current, listOwners);
      edits.acknowledge();
      onSaved(accepted, moved);
    } catch (failure) {
      if (!current()) return;
      if (isPropertyHistoryAccessDenied(failure)) {
        forgetItemReads(client, scope);
        onAccessLost();
        throw failure;
      }
      setCause(failure);
      setConflict(
        issueSourceChanged(failure) || issueConfigurationChanged(failure) || (reloadAfterValidation && !!source.data)
      );
      setError(issueSaveError(failure, "Could not finish the save. Your changes and any pending save are retained."));
      throw failure;
    } finally {
      if (current()) setReconciling(false);
    }
  };
  const reload = async () => {
    const access = captureReadAccess();
    try {
      access();
      const refreshed = await source.refetch({ throwOnError: true });
      access();
      if (edits.isCurrent()) {
        setConflict(false);
        setError(undefined);
        setCause(undefined);
        return refreshed.data;
      }
    } catch (failure) {
      try {
        access();
      } catch {
        return undefined;
      }
      if (edits.isCurrent())
        setError(issueSaveError(failure, "Could not reload the work item. Your changes are retained."));
    }
    return undefined;
  };
  return { source, edits, error, cause, conflict, reload, save, recover: save, busy: edits.isPending || reconciling };
}
