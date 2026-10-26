/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback } from "react";

import { EIssuesStoreType } from "@plane/types";
// hooks
import { useCycleDetails } from "@/hooks/use-cycles";
import { cycleIsCompleted } from "@/queries/cycle-module-selectors";
import { useIssuesActions } from "@/hooks/use-issues-actions";
// components
import { CycleIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseCalendarRoot } from "../base-calendar-root";

export function CycleCalendarLayout() {
  const { workspaceSlug, projectId, cycleId } = useParams({ strict: false });
  const detail = useCycleDetails(workspaceSlug, projectId, cycleId);
  const actions = useIssuesActions(EIssuesStoreType.CYCLE);

  const isCompletedCycle = detail.cycle ? cycleIsCompleted(detail.cycle) : false;

  const addIssuesToView = useCallback(
    async (issueIds: string[]) => {
      if (!workspaceSlug || !projectId || !cycleId) throw new Error();
      return actions.addIssuesToCycle(projectId, cycleId, issueIds);
    },
    [actions, workspaceSlug, projectId, cycleId]
  );

  if (!cycleId) return null;

  return (
    <BaseCalendarRoot
      QuickActions={CycleIssueQuickActions}
      addIssuesToView={addIssuesToView}
      isCompletedCycle={isCompletedCycle}
      viewId={cycleId?.toString()}
    />
  );
}
