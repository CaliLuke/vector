/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// components
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { ProjectStateRoot } from "@/components/project-states";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
// hook
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { StatesProjectSettingsHeader } from "./header";

function StatesSettingsPage(): React.JSX.Element | null {
  const { workspaceSlug, projectId } = Route.useParams();
  // store
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const { can: canProject, isPending, isError } = useProjectPermissions(workspaceSlug, projectId);

  const { t } = useTranslation();

  // derived values
  const pageTitle = currentProjectDetails?.name ? `${currentProjectDetails?.name} - States` : undefined;
  // derived values
  const canPerformProjectMemberActions = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  if (isPending) return null;

  if (isError || !canPerformProjectMemberActions) {
    return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<StatesProjectSettingsHeader />}>
      <PageHead title={pageTitle} />
      <div className="w-full">
        <SettingsHeading
          title={t("project_settings.states.heading")}
          description={t("project_settings.states.description")}
        />
        <div className="mt-6">
          <ProjectStateRoot workspaceSlug={workspaceSlug} projectId={projectId} />
        </div>
      </div>
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/states")({
  component: StatesSettingsPage,
});
