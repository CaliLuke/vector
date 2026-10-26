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
import { useWorkItemDetailScope } from "@/hooks/use-work-item-detail";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import {
  allSubItems,
  captureIssueDetailReconciliation,
  issueDetailKeys,
  issueSubItemsOptions,
} from "@/queries/issue-details";
import {
  captureIssueListMutationOwner,
  patchIssueMutation,
  refreshCapturedIssueLists,
} from "@/queries/issue-list-mutations";
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
  const service = useMemo(() => new IssueService(issueServiceType), [issueServiceType]);
  const mutationStoreType =
    issueServiceType === EIssueServiceType.EPICS ? EIssuesStoreType.EPIC : EIssuesStoreType.PROJECT;
  const resolveListOwner = useIssueListMutationOwner(mutationStoreType);
  const listActions = useIssuesActions(mutationStoreType);

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
      fetchSubIssues: async (targetWorkspace, targetProject, targetParentId) => {
        try {
          const owner = captureOwner(targetParentId);
          assertMutationScope(owner, targetWorkspace, targetProject, true);
          const options = issueSubItemsOptions(client, owner.scope, targetParentId, access.isReady);
          await client.fetchQuery(options);
        } catch {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: t("entity.fetch.failed", {
              entity:
                issueServiceType === EIssueServiceType.ISSUES
                  ? t("common.sub_work_items", { count: 2 })
                  : t("issue.label", { count: 2 }),
            }),
          });
        }
      },
      addSubIssue: async (_workspaceSlug, _projectId, targetParentId, issueIds) => {
        const owner = captureOwner(targetParentId);
        assertMutationScope(owner, _workspaceSlug, _projectId, true);
        const listOwner = await resolveListOwner(_workspaceSlug, _projectId);
        if (!listOwner) throw new Error("Work item access is unavailable. Retry the operation.");
        const capturedList = captureIssueListMutationOwner(client, listOwner);
        const childDetailOwner = captureIssueDetailReconciliation(client, owner.scope, issueIds);
        const response = await service.addSubIssues(_workspaceSlug, _projectId, targetParentId, {
          sub_issue_ids: issueIds,
        });
        const records = allSubItems(response);
        if (owner.isCurrent()) {
          await Promise.all(records.map((record) => childDetailOwner.reconcile(record)));
          await refreshCapturedIssueLists(client, capturedList);
          refresh(owner);
        }
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
      updateSubIssue: async (
        _workspaceSlug,
        _projectId,
        targetParentId,
        issueId,
        issueData,
        _oldIssue = {},
        fromModal = false
      ) => {
        const owner = captureOwner(targetParentId);
        assertMutationScope(owner, _workspaceSlug, _projectId);
        const childDetailOwner = fromModal
          ? undefined
          : captureIssueDetailReconciliation(client, owner.scope, [issueId]);
        try {
          if (!fromModal) {
            const listOwner = await resolveListOwner(_workspaceSlug, _projectId);
            if (!listOwner) throw new Error("Work item access is unavailable. Retry the operation.");
            const receipt = await patchIssueMutation(client, listOwner, issueId, issueData);
            await childDetailOwner?.reconcile(receipt);
          }
          if (childDetailOwner && fromModal) void childDetailOwner.refresh();
          if (owner.isCurrent()) refresh(owner);
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("toast.success"),
            message: t("sub_work_item.update.success"),
          });
        } catch (error) {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: t("sub_work_item.update.error"),
          });
          throw error;
        }
      },
      removeSubIssue: async (_workspaceSlug, _projectId, targetParentId, issueId) => {
        const owner = captureOwner(targetParentId);
        assertMutationScope(owner, _workspaceSlug, _projectId);
        const childDetailOwner = captureIssueDetailReconciliation(client, owner.scope, [issueId]);
        try {
          const listOwner = await resolveListOwner(_workspaceSlug, _projectId);
          if (!listOwner) throw new Error("Work item access is unavailable. Retry the operation.");
          const receipt = await patchIssueMutation(client, listOwner, issueId, { parent_id: null });
          await childDetailOwner.reconcile(receipt);
          if (owner.isCurrent()) refresh(owner);
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("toast.success"),
            message: t("entity.remove.success", {
              entity:
                issueServiceType === EIssueServiceType.ISSUES
                  ? t("common.sub_work_items")
                  : t("issue.label", { count: 1 }),
            }),
          });
        } catch (error) {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: t("entity.remove.failed", {
              entity:
                issueServiceType === EIssueServiceType.ISSUES
                  ? t("common.sub_work_items")
                  : t("issue.label", { count: 1 }),
            }),
          });
          throw error;
        }
      },
      deleteSubIssue: async (_workspaceSlug, _projectId, targetParentId, issueId) => {
        const owner = captureOwner(targetParentId);
        assertMutationScope(owner, _workspaceSlug, _projectId);
        const childDetailOwner = captureIssueDetailReconciliation(client, owner.scope, [issueId]);
        try {
          await listActions.removeIssue(_projectId, issueId);
          childDetailOwner.remove();
          if (owner.isCurrent()) refresh(owner);
        } catch (error) {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: t("entity.delete.failed", {
              entity:
                issueServiceType === EIssueServiceType.ISSUES
                  ? t("common.sub_work_items")
                  : t("issue.label", { count: 1 }),
            }),
          });
          throw error;
        }
      },
    }),
    [
      access.isReady,
      captureOwner,
      refresh,
      assertMutationScope,
      client,
      issueServiceType,
      listActions,
      resolveListOwner,
      service,
      t,
    ]
  );
  return operations;
}
