/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { EIssuesStoreType } from "@plane/types";
import type { TUserPermissions } from "@plane/types";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import { captureItemLineageScope } from "@/queries/work-item-read-scope";
import { permissionKeys } from "@/queries/permissions";
import { refreshIssueActivity } from "@/queries/issue-discussion";
import { refreshItemSchema } from "@/queries/work-item-detail-schema";
import { refreshPropertyHistory } from "@/queries/work-item-property-history";
import { captureIssueListMutationOwner, refreshCapturedIssueLists } from "@/queries/issue-list-mutations";
import type { CapturedIssueListMutationOwner } from "@/queries/issue-list-mutations";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { isItemAccessDenied } from "@/queries/work-item-access";
import { workItemServiceType } from "@/helpers/work-item-service-type";
import type { PublishedItemEditScope } from "@/queries/published-item-edit";
import type { SavedItemEdit } from "@/services/work-item-edits.service";

type CapturedListOwners = {
  source: CapturedIssueListMutationOwner;
  target?: CapturedIssueListMutationOwner;
  targetProjectId: string;
};

/** Capture list query lineage before the editor sends a write, then reconcile its accepted receipt. */
export function useItemEditReconciliation(scope: PublishedItemEditScope, sourceQueryKey: readonly unknown[]) {
  const client = useQueryClient();
  const resolveListOwner = useIssueListMutationOwner(EIssuesStoreType.PROJECT);

  const captureListOwners = async (targetProjectId: string): Promise<CapturedListOwners> => {
    const sourceOwner = await resolveListOwner(scope.workspaceSlug, scope.projectId);
    const targetOwner =
      targetProjectId === scope.projectId ? sourceOwner : await resolveListOwner(scope.workspaceSlug, targetProjectId);
    if (!sourceOwner || !targetOwner) throw new Error("Work item access changed before saving. Retry the operation.");
    return {
      source: captureIssueListMutationOwner(client, sourceOwner),
      target: targetProjectId === scope.projectId ? undefined : captureIssueListMutationOwner(client, targetOwner),
      targetProjectId,
    };
  };

  const reconcile = async (saved: SavedItemEdit, current: () => boolean, captured: CapturedListOwners) => {
    if (!("sequence_id" in saved) || saved.id !== scope.issueId || !saved.project_id)
      throw new Error("Reopen the saved work item to confirm its updated location.");
    if (saved.project_id !== captured.targetProjectId)
      throw new Error("The saved work item moved to an unexpected project. Reopen it to confirm its location.");

    const roles = client.getQueryData<Record<string, TUserPermissions>>(
      permissionKeys.workspaceProjectRoles(
        scope.accountId,
        scope.workspaceSlug,
        scope.membershipRole,
        scope.membershipRevision
      )
    );
    const destination = {
      ...scope,
      projectId: saved.project_id,
      projectRole: effectiveProjectRole(scope.membershipRole, roles, saved.project_id),
    };
    const destinationCurrent = captureItemLineageScope(client, destination, sourceQueryKey);
    const canApply = () => current() && destinationCurrent();
    if (!canApply()) throw new Error("Work item access changed before refreshing its saved values.");

    const activityAccess = captureProjectAccessScope(client, destination);
    const activityTarget = {
      workspaceSlug: scope.workspaceSlug,
      projectId: saved.project_id,
      issueId: saved.id,
      serviceType: workItemServiceType(saved),
    };
    const activityOwner = {
      accountId: scope.accountId,
      generation: scope.generation,
      isCurrent: () => activityAccess.isCurrent() && !isItemAccessDenied(client, { ...scope, ...activityTarget }),
    };
    const moved = saved.project_id !== scope.projectId;

    await refreshCapturedIssueLists(client, captured.source);
    if (captured.target) await refreshCapturedIssueLists(client, captured.target);
    void Promise.allSettled([
      ...(!moved
        ? [refreshItemSchema(client, scope.workspaceSlug, saved.id), refreshPropertyHistory(client, scope)]
        : []),
      refreshIssueActivity(client, activityTarget, activityOwner),
    ]);
    return moved;
  };

  return { captureListOwners, reconcile };
}
