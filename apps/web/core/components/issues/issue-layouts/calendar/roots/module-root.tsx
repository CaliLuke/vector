/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback } from "react";

// plane imports
import { EIssuesStoreType } from "@plane/types";
// hooks
import { useIssuesActions } from "@/hooks/use-issues-actions";
// local imports
import { ModuleIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseCalendarRoot } from "../base-calendar-root";

export function ModuleCalendarLayout() {
  const { workspaceSlug, projectId, moduleId } = useParams({ strict: false });
  const actions = useIssuesActions(EIssuesStoreType.MODULE);

  const addIssuesToView = useCallback(
    async (issueIds: string[]) => {
      if (!workspaceSlug || !projectId || !moduleId) throw new Error();
      return actions.addIssuesToModule(projectId, moduleId, issueIds);
    },
    [actions, workspaceSlug, projectId, moduleId]
  );

  if (!moduleId) return null;

  return (
    <BaseCalendarRoot QuickActions={ModuleIssueQuickActions} addIssuesToView={addIssuesToView} viewId={moduleId} />
  );
}
