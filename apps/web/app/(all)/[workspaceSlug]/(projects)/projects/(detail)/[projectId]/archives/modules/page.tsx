/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

// components
import { PageHead } from "@/components/core/page-title";
import { ArchivedModuleLayoutRoot, ArchivedModulesHeader } from "@/components/modules";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

function ProjectArchivedModulesPage(): React.JSX.Element | null {
  // router
  const { projectId } = Route.useParams();
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  // derived values
  const project = projectRows?.find((candidate) => candidate.id === projectId);
  const pageTitle = project?.name && `${project?.name} - Archived modules`;

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="relative flex h-full w-full flex-col overflow-hidden">
        <ArchivedModulesHeader />
        <ArchivedModuleLayoutRoot />
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/archives/modules/")({
  component: ProjectArchivedModulesPage,
});
