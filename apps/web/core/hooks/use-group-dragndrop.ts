/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useQueryClient } from "@tanstack/react-query";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { EIssuesStoreType, TIssue, TIssueGroupByOptions, TIssueOrderByOptions } from "@plane/types";
import type { IssueListGroupedIds } from "@/queries/issue-lists";
import type { GroupDropLocation } from "@/components/issues/issue-layouts/utils";
import { handleGroupDragDrop } from "@/components/issues/issue-layouts/utils";
import { ISSUE_FILTER_DEFAULT_DATA } from "@/helpers/issue-list/properties";
import { useIssuesActions } from "./use-issues-actions";
import { useIssueFilterPreferences } from "./use-issue-filter-preferences";
import { useParams } from "@tanstack/react-router";
import {
  getVisibleGroupedIssueIds,
  isNativeDisplayOrder,
} from "@/components/issues/issue-layouts/issue-display-ordering";

type DNDStoreType =
  | EIssuesStoreType.PROJECT
  | EIssuesStoreType.MODULE
  | EIssuesStoreType.CYCLE
  | EIssuesStoreType.PROJECT_VIEW
  | EIssuesStoreType.PROFILE
  | EIssuesStoreType.ARCHIVED
  | EIssuesStoreType.TEAM
  | EIssuesStoreType.TEAM_VIEW
  | EIssuesStoreType.EPIC
  | EIssuesStoreType.TEAM_PROJECT_WORK_ITEMS;

export const useGroupIssuesDragNDrop = (
  storeType: DNDStoreType,
  orderBy: TIssueOrderByOptions | undefined,
  groupBy: TIssueGroupByOptions | undefined,
  subGroupBy?: TIssueGroupByOptions,
  projectedIssueGroups?: IssueListGroupedIds,
  issuesById: Readonly<Record<string, TIssue>> = {},
  canonicalIssueGroups: IssueListGroupedIds | undefined = projectedIssueGroups
) => {
  const route = useParams({ strict: false });
  const { workspaceSlug } = route;
  const client = useQueryClient(),
    accountId = useCurrentUserId(),
    generation = useIdentityGeneration(client);
  const current = useCommittedScope(
    JSON.stringify([
      accountId,
      generation,
      storeType,
      workspaceSlug,
      route.projectId,
      route.cycleId,
      route.moduleId,
      route.viewId,
    ])
  );

  const getIssueById = (issueId: string) => issuesById[issueId];
  const { requestInput } = useIssueFilterPreferences(storeType);
  const { updateIssue, addCycleToIssue, removeCycleFromIssue, changeModulesInIssue } = useIssuesActions(
    storeType,
    requestInput
  );

  const getVisibleIssueIds = (groupId?: string, subGroupId?: string) =>
    getVisibleGroupedIssueIds(
      isNativeDisplayOrder(orderBy) ? projectedIssueGroups : canonicalIssueGroups,
      getIssueById,
      groupId,
      subGroupId
    );

  /**
   * update Issue on Drop, checks if modules or cycles are changed and then calls appropriate functions
   * @param projectId
   * @param issueId
   * @param data
   * @param issueUpdates
   */
  const updateIssueOnDrop = async (
    projectId: string,
    issueId: string,
    data: Partial<TIssue>,
    issueUpdates: {
      [groupKey: string]: {
        ADD: string[];
        REMOVE: string[];
      };
    }
  ) => {
    const errorToastProps = {
      type: TOAST_TYPE.ERROR,
      title: "Error!",
      message: "Error while updating work item",
    };
    const showError = () => {
      if (current()) setToast(errorToastProps);
    };
    const moduleKey = ISSUE_FILTER_DEFAULT_DATA["module"];
    const cycleKey = ISSUE_FILTER_DEFAULT_DATA["cycle"];

    const isModuleChanged = Object.keys(data).includes(moduleKey);
    const isCycleChanged = Object.keys(data).includes(cycleKey);

    if (isCycleChanged && workspaceSlug) {
      if (data[cycleKey]) {
        await addCycleToIssue(projectId, data[cycleKey]?.toString() ?? "", issueId).catch(showError);
      } else {
        const cycleId = getIssueById(issueId)?.cycle_id;
        if (cycleId) await removeCycleFromIssue(projectId, cycleId, issueId).catch(showError);
      }
      delete data[cycleKey];
    }

    if (isModuleChanged && workspaceSlug && issueUpdates[moduleKey]) {
      await changeModulesInIssue(projectId, issueId, issueUpdates[moduleKey].ADD, issueUpdates[moduleKey].REMOVE).catch(
        showError
      );
      delete data[moduleKey];
    }

    if (updateIssue && Object.keys(data).length > 0) await updateIssue(projectId, issueId, data).catch(showError);
  };

  const handleOnDrop = async (source: GroupDropLocation, destination: GroupDropLocation) => {
    if (
      source.columnId &&
      destination.columnId &&
      destination.columnId === source.columnId &&
      destination.id === source.id
    )
      return;

    await handleGroupDragDrop(
      source,
      destination,
      getIssueById,
      getVisibleIssueIds,
      updateIssueOnDrop,
      groupBy,
      subGroupBy,
      orderBy !== "sort_order"
    ).catch((err) => {
      if (!current()) return;
      setToast({
        title: "Error!",
        type: TOAST_TYPE.ERROR,
        message:
          typeof err === "object" && err !== null && "detail" in err && typeof err.detail === "string"
            ? err.detail
            : "Failed to perform this action",
      });
    });
  };

  return handleOnDrop;
};
