/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { ParentIssueMobileHeader } from "@/components/issues/issue-layouts/filters/header/parent-issue-mobile-header";
import { useModuleDetails } from "@/hooks/use-modules";
import { useRouteProject } from "@/hooks/use-projects";
import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";

export const ModuleIssuesMobileHeader = function ModuleIssuesMobileHeader() {
  const { workspaceSlug, projectId, moduleId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/modules/$moduleId",
  });
  const { project: projectDetails } = useRouteProject();
  const { setDisplay } = useParentIssuePreferences();
  const { module } = useModuleDetails(workspaceSlug, projectId, moduleId);

  return (
    <ParentIssueMobileHeader
      workspaceSlug={workspaceSlug}
      projectDetails={projectDetails}
      moduleDetails={module ?? undefined}
      canUpdateLayout={Boolean(workspaceSlug && projectId)}
      setDisplay={setDisplay}
    />
  );
};
