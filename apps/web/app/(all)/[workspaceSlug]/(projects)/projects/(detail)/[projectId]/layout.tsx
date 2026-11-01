/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Outlet } from "@tanstack/react-router";
// plane imports
import { ProjectTabHeader } from "@/components/navigation/project-tab-header";
// components
// hooks
// layouts
import { ProjectAuthWrapper } from "@/layouts/auth-layout/project-wrapper";
// local imports

function ProjectLayout(): React.JSX.Element | null {
  // router
  const { workspaceSlug, projectId } = Route.useParams();
  return (
    <>
      <ProjectTabHeader
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        leftItemClassName="flex h-full max-w-full items-center gap-2"
      />
      <ProjectAuthWrapper workspaceSlug={workspaceSlug} projectId={projectId}>
        <Outlet />
      </ProjectAuthWrapper>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId")({
  component: ProjectLayout,
});
