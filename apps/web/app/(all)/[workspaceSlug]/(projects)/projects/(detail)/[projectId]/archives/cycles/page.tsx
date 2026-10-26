/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

// components
import { PageHead } from "@/components/core/page-title";
import { ArchivedCycleLayoutRoot } from "@/components/cycles/archived-cycles";
import { ArchivedCyclesHeader } from "@/components/cycles/archived-cycles/header";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

function ProjectArchivedCyclesPage(): React.JSX.Element | null {
  // router
  const { projectId } = Route.useParams();
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  // derived values
  const project = projectRows?.find((candidate) => candidate.id === projectId);
  const pageTitle = project?.name && `${project?.name} - Archived cycles`;

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="relative flex h-full w-full flex-col overflow-hidden">
        <ArchivedCyclesHeader />
        <ArchivedCycleLayoutRoot />
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/archives/cycles/")({
  component: ProjectArchivedCyclesPage,
});
