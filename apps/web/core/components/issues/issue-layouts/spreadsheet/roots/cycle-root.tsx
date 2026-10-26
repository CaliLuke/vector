/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React, { useCallback } from "react";

// hooks
import { EUserPermissions } from "@plane/constants";
import { useCycleDetails } from "@/hooks/use-cycles";
import { cycleIsCompleted } from "@/queries/cycle-module-selectors";
import { useProjectPermissions } from "@/hooks/use-permissions";
// components
import { CycleIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseSpreadsheetRoot } from "../base-spreadsheet-root";

export const CycleSpreadsheetLayout = function CycleSpreadsheetLayout() {
  // router
  const { cycleId } = useParams({ strict: false });
  // store hooks
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const detail = useCycleDetails(workspaceSlug, projectId, cycleId);
  // auth
  const isCompletedCycle = detail.cycle ? cycleIsCompleted(detail.cycle) : false;
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const canEditIssueProperties = useCallback(
    () => !isCompletedCycle && isEditingAllowed,
    [isCompletedCycle, isEditingAllowed]
  );

  if (!cycleId) return null;

  return (
    <BaseSpreadsheetRoot
      QuickActions={CycleIssueQuickActions}
      canEditPropertiesBasedOnProject={canEditIssueProperties}
      isCompletedCycle={isCompletedCycle}
      viewId={cycleId.toString()}
    />
  );
};
