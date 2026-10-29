/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TIssueServiceType, TSubIssueOperations } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
import { copyUrlToClipboard } from "@plane/utils/string";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useWorkItemDetailScope } from "@/hooks/use-work-item-detail";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import { allSubItems, captureIssueDetailReconciliation, issueDetailKeys } from "@/queries/issue-details";
import { captureIssueListMutationOwner, refreshCapturedIssueLists } from "@/queries/issue-list-mutations";
import { currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { getIdentitySessionGeneration } from "@/queries/identity";
import { itemAccessOptions } from "@/queries/work-item-access";
import { IssueService } from "@/services/issue";

export function useSubIssueOperations(
  workspaceSlug: string,
  projectId: string,
  parentIssueId: string,
  issueServiceType: TIssueServiceType
): TSubIssueOperations {
  const { t } = useTranslation();
  const client = useQueryClient();
  const access = useWorkItemDetailScope(workspaceSlug, projectId, issueServiceType, parentIssueId);
  const isUiCurrent = useCommittedScope(JSON.stringify([workspaceSlug, projectId, parentIssueId, issueServiceType]));
  const service = useMemo(() => new IssueService(issueServiceType), [issueServiceType]);
  const parentStoreType =
    issueServiceType === EIssueServiceType.EPICS ? EIssuesStoreType.EPIC : EIssuesStoreType.PROJECT;
  const resolveParentListOwner = useIssueListMutationOwner(parentStoreType);
  const childActions = useIssuesActions(EIssuesStoreType.PROJECT);

  const captureOwner = useCallback(
    (targetParentId: string) => {
      const scope = access.scope;
      const queryKey = issueDetailKeys.subItems(scope, targetParentId);
      const capturedQuery = client.getQueryCache().find({ queryKey, exact: true });
      const projectScope = captureProjectAccessScope(client, scope);
      const itemScope = {
        accountId: scope.accountId,
        generation: scope.sessionGeneration,
        workspaceSlug: scope.workspaceSlug,
        projectId: scope.projectId,
        issueId: targetParentId,
        serviceType: scope.serviceType,
      };
      const itemAccessKey = itemAccessOptions(itemScope).queryKey;
      const capturedItemAccess = client.getQueryCache().find({ queryKey: itemAccessKey, exact: true });
      const itemAccessRevision = capturedItemAccess?.state.dataUpdatedAt;
      return {
        scope,
        queryKey,
        parentId: targetParentId,
        isCurrent: () =>
          access.isReady &&
          client.getQueryCache().find({ queryKey, exact: true }) === capturedQuery &&
          client.getQueryCache().find({ queryKey: itemAccessKey, exact: true }) === capturedItemAccess &&
          capturedItemAccess?.state.dataUpdatedAt === itemAccessRevision &&
          capturedItemAccess?.state.data !== true &&
          getIdentitySessionGeneration(client) === scope.sessionGeneration &&
          projectScope.isVerified(),
      };
    },
    [access.isReady, access.scope, client]
  );

  const refresh = useCallback(
    (owner: ReturnType<typeof captureOwner>) => {
      if (!owner.isCurrent()) return;
      void client.invalidateQueries({ queryKey: owner.queryKey, exact: true }).catch(() => undefined);
      void client
        .invalidateQueries({ queryKey: issueDetailKeys.detail(owner.scope, owner.parentId), exact: true })
        .catch(() => undefined);
    },
    [client]
  );
  const assertMutationScope = useCallback(
    (
      owner: ReturnType<typeof captureOwner>,
      targetWorkspace: string,
      targetProject: string,
      requireParentProject = false
    ) => {
      if (
        !owner.isCurrent() ||
        owner.scope.workspaceSlug !== targetWorkspace ||
        (requireParentProject && owner.scope.projectId !== targetProject)
      )
        throw new Error("Work item access changed. Reload before continuing.");
    },
    []
  );

  const captureChildDetails = useCallback(
    (targetProjectId: string, issueIds: readonly string[]) => {
      const { workspaceRole, ...scope } = currentParentCatalogueScope(client, workspaceSlug, targetProjectId).project;
      return captureIssueDetailReconciliation(
        client,
        { ...scope, membershipRole: workspaceRole, serviceType: EIssueServiceType.ISSUES },
        issueIds
      );
    },
    [client, workspaceSlug]
  );

  const updateChild = useCallback<TSubIssueOperations["updateSubIssue"]>(
    async (targetWorkspace, targetProject, targetParentId, issueId, data) => {
      const owner = captureOwner(targetParentId);
      assertMutationScope(owner, targetWorkspace, targetProject);
      const childDetailOwner = captureChildDetails(targetProject, [issueId]);
      if (!childActions.updateIssue) throw new Error("Work item editing is unavailable.");
      const receipt = await childActions.updateIssue(targetProject, issueId, data);
      await childDetailOwner.reconcile(receipt);
      refresh(owner);
    },
    [captureOwner, assertMutationScope, captureChildDetails, childActions, refresh]
  );

  const operations: TSubIssueOperations = useMemo(
    () => ({
      copyLink: async (path) => {
        await copyUrlToClipboard(path);
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("common.link_copied"),
          message: t("entity.link_copied_to_clipboard", {
            entity:
              issueServiceType === EIssueServiceType.ISSUES
                ? t("common.sub_work_items", { count: 1 })
                : t("issue.label", { count: 1 }),
          }),
        });
      },
      addSubIssue: async (_workspaceSlug, _projectId, targetParentId, issues) => {
        const issueIds = issues.map((issue) => issue.id);
        const owner = captureOwner(targetParentId);
        assertMutationScope(owner, _workspaceSlug, _projectId, true);
        const listOwner = await resolveParentListOwner(_workspaceSlug, _projectId);
        if (!listOwner) throw new Error("Work item access is unavailable. Retry the operation.");
        const capturedList = captureIssueListMutationOwner(client, listOwner);
        const childDetails = issues.map((issue) => ({
          issue,
          owner: captureChildDetails(issue.project_id, [issue.id]),
        }));
        const response = await service.addSubIssues(_workspaceSlug, _projectId, targetParentId, {
          sub_issue_ids: issueIds,
        });
        const records = allSubItems(response);
        if (owner.isCurrent()) {
          await Promise.all(
            childDetails.map(({ issue, owner: childOwner }) => {
              const saved = records.find((record) => record.id === issue.id && record.project_id === issue.project_id);
              return saved ? childOwner.reconcile(saved) : childOwner.refresh();
            })
          );
          await refreshCapturedIssueLists(client, capturedList);
          refresh(owner);
        }
        if (!owner.isCurrent() || !isUiCurrent()) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("entity.add.success", {
            entity:
              issueServiceType === EIssueServiceType.ISSUES
                ? t("common.sub_work_items")
                : t("issue.label", { count: issueIds.length }),
          }),
        });
      },
      refreshSubIssues: (targetParentId) => refresh(captureOwner(targetParentId)),
      updateSubIssue: updateChild,
      removeSubIssue: (targetWorkspace, targetProject, targetParentId, issueId) =>
        updateChild(targetWorkspace, targetProject, targetParentId, issueId, { parent_id: null }),
      deleteSubIssue: async (_workspaceSlug, _projectId, targetParentId, issueId) => {
        const owner = captureOwner(targetParentId);
        assertMutationScope(owner, _workspaceSlug, _projectId);
        const childDetailOwner = captureChildDetails(_projectId, [issueId]);
        await childActions.removeIssue(_projectId, issueId);
        childDetailOwner.remove();
        refresh(owner);
      },
    }),
    [
      captureOwner,
      captureChildDetails,
      isUiCurrent,
      refresh,
      assertMutationScope,
      client,
      issueServiceType,
      childActions,
      resolveParentListOwner,
      updateChild,
      service,
      t,
    ]
  );
  return operations;
}
