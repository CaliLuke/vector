/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
// components
import { PageHead } from "@/components/core/page-title";
import { ArchivedIssuesHeader } from "@/components/issues/archived-issues-header";
import { ArchivedIssueLayoutRoot } from "@/components/issues/issue-layouts/roots/archived-issue-layout-root";
import { EIssuesStoreType } from "@plane/types";
import { LocalIssuePreferencesProvider } from "@/providers/local-issue-preferences";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

function ProjectArchivedIssuesPage(): React.JSX.Element | null {
  // router
  const { workspaceSlug, projectId } = Route.useParams();
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  // derived values
  const project = projectRows?.find((candidate) => candidate.id === projectId);
  const pageTitle = project?.name && `${project?.name} - Archived work items`;

  return (
    <LocalIssuePreferencesProvider
      domain={EIssuesStoreType.ARCHIVED}
      workspaceSlug={workspaceSlug}
      entityId={projectId}
    >
      <>
        <PageHead title={pageTitle} />
        <div className="relative flex h-full w-full flex-col overflow-hidden">
          <ArchivedIssuesHeader />
          <ArchivedIssueLayoutRoot />
        </div>
      </>
    </LocalIssuePreferencesProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/archives/issues/")({
  component: ProjectArchivedIssuesPage,
});
