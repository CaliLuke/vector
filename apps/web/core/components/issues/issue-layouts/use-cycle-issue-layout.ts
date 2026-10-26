/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback } from "react";
import { EUserPermissions } from "@plane/constants";
import { EIssuesStoreType } from "@plane/types";
import { useCycleDetails } from "@/hooks/use-cycles";
import { cycleIsCompleted } from "@/queries/cycle-module-selectors";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { CycleIssueQuickActions } from "./quick-action-dropdowns";

export function useCycleIssueLayout() {
  const { workspaceSlug, projectId, cycleId } = useParams({ strict: false });
  const detail = useCycleDetails(workspaceSlug, projectId, cycleId);
  const actions = useIssuesActions(EIssuesStoreType.CYCLE);
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const isCompletedCycle = detail.cycle ? cycleIsCompleted(detail.cycle) : false;
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const canEditIssueProperties = useCallback(
    () => !isCompletedCycle && isEditingAllowed,
    [isCompletedCycle, isEditingAllowed]
  );

  const addIssuesToView = useCallback(
    async (issueIds: string[]) => {
      if (!workspaceSlug || !projectId || !cycleId) throw new Error();
      return actions.addIssuesToCycle(projectId, cycleId, issueIds);
    },
    [actions, workspaceSlug, projectId, cycleId]
  );

  return {
    QuickActions: CycleIssueQuickActions,
    addIssuesToView,
    canEditPropertiesBasedOnProject: canEditIssueProperties,
    isCompletedCycle,
    viewId: cycleId,
  };
}
