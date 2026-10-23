/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { observer } from "mobx-react";
// components
import { PageHead } from "@/components/core/page-title";
import { ArchivedIssuesHeader } from "@/components/issues/archived-issues-header";
import { ArchivedIssueLayoutRoot } from "@/components/issues/issue-layouts/roots/archived-issue-layout-root";
// hooks
import { useProject } from "@/hooks/store/use-project";

function ProjectArchivedIssuesPage(): React.JSX.Element | null {
  // router
  const { projectId } = Route.useParams();
  // store hooks
  const { getProjectById } = useProject();
  // derived values
  const project = getProjectById(projectId);
  const pageTitle = project?.name && `${project?.name} - Archived work items`;

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="relative flex h-full w-full flex-col overflow-hidden">
        <ArchivedIssuesHeader />
        <ArchivedIssueLayoutRoot />
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/archives/issues/")({
  component: observer(ProjectArchivedIssuesPage),
});
