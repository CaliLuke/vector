/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IProject } from "@plane/types";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { AutoArchiveAutomation, AutoCloseAutomation } from "@/components/automation";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
// hooks
import { useProjectMutations, useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { AutomationsProjectSettingsHeader } from "./header";

function AutomationSettingsPage(): React.JSX.Element | null {
  // router
  const { workspaceSlug, projectId } = Route.useParams();
  // store hooks
  const projectQuery = useRouteProject();
  const projectDetails = projectQuery.project;
  const updateProject = useProjectMutations(workspaceSlug, projectId);
  const { can: canProject, isPending, isError } = useProjectPermissions(workspaceSlug, projectId);

  const { t } = useTranslation();

  // derived values
  const canPerformProjectAdminActions = canProject([EUserPermissions.ADMIN]);

  const handleChange = async (formData: Partial<IProject>) => {
    if (!projectDetails) return;

    try {
      await updateProject.mutateAsync({ type: "update", projectId, data: formData });
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Something went wrong. Please try again.",
      });
    }
  };

  // derived values
  const pageTitle = projectDetails?.name ? `${projectDetails?.name} - Automations` : undefined;

  if (isPending || projectQuery.isPending) return null;

  if (isError || projectQuery.isError || projectQuery.accessDenied || !canPerformProjectAdminActions) {
    return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<AutomationsProjectSettingsHeader />} hugging>
      <PageHead title={pageTitle} />
      <section className={`w-full ${canPerformProjectAdminActions ? "" : "opacity-60"}`}>
        <SettingsHeading
          title={t("project_settings.automations.heading")}
          description={t("project_settings.automations.description")}
        />
        <div className="mt-6">
          <AutoArchiveAutomation handleChange={handleChange} />
          <AutoCloseAutomation handleChange={handleChange} />
        </div>
      </section>
    </SettingsContentWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/projects/$projectId/automations/")({
  component: AutomationSettingsPage,
});
