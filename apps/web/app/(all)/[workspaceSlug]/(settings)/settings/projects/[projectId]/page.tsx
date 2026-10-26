/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

// plane imports
import { EUserPermissions } from "@plane/constants";
// components
import { PageHead } from "@/components/core/page-title";
import { ProjectDetailsForm } from "@/components/project/form";
import { ProjectDetailsFormLoader } from "@/components/project/form-loader";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
// hooks
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { GeneralProjectSettingsHeader } from "./header";
import { GeneralProjectSettingsControlSection } from "@/components/project/settings/control-section";

function ProjectSettingsPage(): React.JSX.Element | null {
  // router
  const { workspaceSlug, projectId } = Route.useParams();
  // store hooks
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const isAdmin = canProject([EUserPermissions.ADMIN]);

  const pageTitle = currentProjectDetails?.name ? `${currentProjectDetails?.name} - General Settings` : undefined;

  return (
    <SettingsContentWrapper header={<GeneralProjectSettingsHeader />}>
      <PageHead title={pageTitle} />
      <div className={`w-full ${isAdmin ? "" : "opacity-60"}`}>
        {currentProjectDetails ? (
          <ProjectDetailsForm
            project={currentProjectDetails}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            isAdmin={isAdmin}
          />
        ) : (
          <ProjectDetailsFormLoader />
        )}
        {isAdmin && <GeneralProjectSettingsControlSection projectId={projectId} />}
      </div>
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/")({
  component: ProjectSettingsPage,
});
