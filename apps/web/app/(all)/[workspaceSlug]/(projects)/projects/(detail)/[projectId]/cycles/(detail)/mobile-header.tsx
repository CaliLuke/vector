/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { ParentIssueMobileHeader } from "@/components/issues/issue-layouts/filters/header/parent-issue-mobile-header";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useRouteProject } from "@/hooks/use-projects";
import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";

export const CycleIssuesMobileHeader = function CycleIssuesMobileHeader() {
  const { workspaceSlug, projectId, cycleId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/cycles/$cycleId",
  });
  const { project: projectDetails } = useRouteProject();
  const { setDisplay } = useParentIssuePreferences();
  const { cycle } = useCycleDetails(workspaceSlug, projectId, cycleId);

  return (
    <ParentIssueMobileHeader
      workspaceSlug={workspaceSlug}
      projectDetails={projectDetails}
      cycleDetails={cycle ?? undefined}
      canUpdateLayout={Boolean(workspaceSlug && projectId && cycleId)}
      setDisplay={setDisplay}
      showLayoutLabel
    />
  );
};
