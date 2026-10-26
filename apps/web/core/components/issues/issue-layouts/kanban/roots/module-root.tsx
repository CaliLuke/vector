/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

// plane imports
import { EIssuesStoreType } from "@plane/types";
// hooks
import { useIssuesActions } from "@/hooks/use-issues-actions";
// local imports
import { ModuleIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseKanBanRoot } from "../base-kanban-root";

export function ModuleKanBanLayout() {
  const { workspaceSlug, projectId, moduleId } = useParams({ strict: false });
  const actions = useIssuesActions(EIssuesStoreType.MODULE);

  return (
    <BaseKanBanRoot
      QuickActions={ModuleIssueQuickActions}
      addIssuesToView={async (issueIds: string[]) => {
        if (!workspaceSlug || !projectId || !moduleId) throw new Error();
        return actions.addIssuesToModule(projectId, moduleId, issueIds);
      }}
      viewId={moduleId?.toString()}
    />
  );
}
