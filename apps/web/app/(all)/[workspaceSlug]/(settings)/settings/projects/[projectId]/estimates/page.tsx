/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

// components
import { ProjectSettingsAccess } from "@/components/settings/project/access";
import { PageHead } from "@/components/core/page-title";
import { EstimateRoot } from "@/components/estimates";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
// hooks
import { useRouteProject } from "@/hooks/use-projects";
// local imports
import { EstimatesProjectSettingsHeader } from "./header";

function EstimatesSettingsPage(): React.JSX.Element | null {
  const { workspaceSlug, projectId } = Route.useParams();
  // store
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;

  // derived values
  const pageTitle = currentProjectDetails?.name ? `${currentProjectDetails?.name} - Estimates` : undefined;
  return (
    <ProjectSettingsAccess workspaceSlug={workspaceSlug} projectId={projectId}>
      <SettingsContentWrapper header={<EstimatesProjectSettingsHeader />}>
        <PageHead title={pageTitle} />
        <div className="w-full">
          <EstimateRoot workspaceSlug={workspaceSlug} projectId={projectId} isAdmin />
        </div>
      </SettingsContentWrapper>
    </ProjectSettingsAccess>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/estimates")({
  component: EstimatesSettingsPage,
});
