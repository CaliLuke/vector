/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";

// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// components
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { ProjectMemberList } from "@/components/project/member-list";
import { ProjectSettingsMemberDefaults } from "@/components/project/project-settings-member-defaults";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
// hooks
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions, useWorkspacePermissions } from "@/hooks/use-permissions";
// local imports
import { MembersProjectSettingsHeader } from "./header";

function MembersSettingsPage(): React.JSX.Element | null {
  // router
  const { workspaceSlug, projectId } = Route.useParams();
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const {
    can: canProject,
    isPending: isProjectPending,
    isError: isProjectError,
  } = useProjectPermissions(workspaceSlug, projectId);
  const {
    can: canWorkspace,
    isPending: isWorkspacePending,
    isError: isWorkspaceError,
  } = useWorkspacePermissions(workspaceSlug);
  // derived values
  const pageTitle = currentProjectDetails?.name ? `${currentProjectDetails?.name} - Members` : undefined;
  const isProjectMemberOrAdmin = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  const isWorkspaceAdmin = canWorkspace([EUserPermissions.ADMIN]);
  const canPerformProjectMemberActions = isProjectMemberOrAdmin || isWorkspaceAdmin;

  if (isWorkspacePending || (!isWorkspaceAdmin && isProjectPending)) return null;

  if (isWorkspaceError || (!isWorkspaceAdmin && isProjectError) || !canPerformProjectMemberActions) {
    return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<MembersProjectSettingsHeader />} hugging>
      <PageHead title={pageTitle} />
      <SettingsHeading title={t("common.members")} />
      <ProjectSettingsMemberDefaults projectId={projectId} workspaceSlug={workspaceSlug} />
      <ProjectMemberList projectId={projectId} workspaceSlug={workspaceSlug} />
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/members")({
  component: MembersSettingsPage,
});
