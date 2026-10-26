/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";

// assets
import emptyView from "@/app/assets/empty-state/view.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
import { ProjectViewLayoutRoot } from "@/components/issues/issue-layouts/roots/project-view-layout-root";
import { Button } from "@plane/propel/button";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectView } from "@/hooks/use-project-views";

function ProjectViewIssuesPage(): React.JSX.Element | null {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId, viewId } = Route.useParams();
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const viewQuery = useProjectView(workspaceSlug, projectId, viewId);
  // derived values
  const projectView = viewQuery.view;
  const project = projectRows?.find((candidate) => candidate.id === projectId);
  const pageTitle = project?.name && projectView?.name ? `${project?.name} - ${projectView?.name}` : undefined;

  if (viewQuery.isError || viewQuery.isDenied) {
    return (
      <EmptyState
        image={emptyView}
        title={viewQuery.isDenied ? "View is unavailable" : "Unable to load view"}
        description={
          viewQuery.isDenied
            ? "The view you are looking for does not exist or you don't have permission to view it."
            : "The project view could not be loaded. Retry or return to the views list."
        }
        primaryButton={{
          text: "View other views",
          onClick: () => void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/views` }),
        }}
        secondaryButton={
          viewQuery.isError ? (
            <Button variant="secondary" disabled={viewQuery.isFetching} onClick={() => void viewQuery.retry()}>
              Retry
            </Button>
          ) : undefined
        }
      />
    );
  }

  if (viewQuery.isPending || !projectView) return null;

  return (
    <>
      <PageHead title={pageTitle} />
      <ProjectViewLayoutRoot />
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/views/$viewId/")({
  component: ProjectViewIssuesPage,
});
