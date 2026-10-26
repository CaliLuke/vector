/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import type { TIssue } from "@plane/types";
import { useQueryClient } from "@tanstack/react-query";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { issueListMutation, patchIssueMutation } from "@/queries/issue-list-mutations";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import { useWorkItemByIdentifier } from "@/hooks/use-work-item-detail";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

type TMutationOwnerResolver = ReturnType<typeof useIssueListMutationOwner>;
type TEntityDetailScope = ReturnType<typeof useWorkItemByIdentifier>["scope"];
type TProjectRows = ReturnType<typeof useWorkspaceProjectDetails>["projects"];

export function workItemProjectDetails(entityDetails: TIssue | null | undefined, projectRows: TProjectRows) {
  const project = entityDetails?.project_id
    ? projectRows?.find((row) => row.id === entityDetails.project_id)
    : undefined;
  return { projectDetails: project, isEstimateEnabled: Boolean(project?.estimate) };
}

export function useWorkItemEntityMutationActions(
  client: ReturnType<typeof useQueryClient>,
  detailScope: TEntityDetailScope,
  resolveProjectListOwner: TMutationOwnerResolver
) {
  const refreshDetailAfter = useCallback(
    async (workspace: string, project: string, issueId: string, mutation: Parameters<typeof issueListMutation>[2]) => {
      const detailOwner = captureIssueDetailReconciliation(client, detailScope, [issueId]);
      const listOwner = await resolveProjectListOwner(workspace, project);
      if (!listOwner) throw new Error("Work item access is unavailable. Retry the operation.");
      await issueListMutation(client, listOwner, mutation);
      await detailOwner.refresh();
    },
    [client, detailScope, resolveProjectListOwner]
  );
  const updateEntity = useCallback(
    async (workspace: string, project: string, issueId: string, data: Partial<TIssue>) => {
      const detailOwner = captureIssueDetailReconciliation(client, detailScope, [issueId]);
      const listOwner = await resolveProjectListOwner(workspace, project);
      if (!listOwner) throw new Error("Work item access is unavailable. Retry the operation.");
      const receipt = await patchIssueMutation(client, listOwner, issueId, data);
      await detailOwner.reconcile(receipt);
      return receipt;
    },
    [client, detailScope, resolveProjectListOwner]
  );
  const addCycleToIssue = useCallback(
    (workspace: string, project: string, cycleId: string, issueId: string) =>
      refreshDetailAfter(workspace, project, issueId, { kind: "cycle-add", cycleId, issueIds: [issueId] }),
    [refreshDetailAfter]
  );
  const removeIssueFromCycle = useCallback(
    (workspace: string, project: string, cycleId: string, issueId: string) =>
      refreshDetailAfter(workspace, project, issueId, { kind: "cycle-remove", cycleId, issueId }),
    [refreshDetailAfter]
  );
  const changeModulesInIssue = useCallback(
    (workspace: string, project: string, issueId: string, added: string[], removed: string[]) =>
      refreshDetailAfter(workspace, project, issueId, { kind: "module-change", issueId, added, removed }),
    [refreshDetailAfter]
  );
  const addCycleToEpic = useCallback(
    (workspace: string, project: string, cycleId: string, issueId: string) =>
      refreshDetailAfter(workspace, project, issueId, { kind: "cycle-add", cycleId, issueIds: [issueId] }),
    [refreshDetailAfter]
  );
  const removeEpicFromCycle = useCallback(
    (workspace: string, project: string, cycleId: string, issueId: string) =>
      refreshDetailAfter(workspace, project, issueId, { kind: "cycle-remove", cycleId, issueId }),
    [refreshDetailAfter]
  );
  const changeModulesInEpic = useCallback(
    (workspace: string, project: string, issueId: string, added: string[], removed: string[]) =>
      refreshDetailAfter(workspace, project, issueId, { kind: "module-change", issueId, added, removed }),
    [refreshDetailAfter]
  );

  return {
    updateEntity,
    addCycleToIssue,
    removeIssueFromCycle,
    changeModulesInIssue,
    addCycleToEpic,
    removeEpicFromCycle,
    changeModulesInEpic,
  };
}
