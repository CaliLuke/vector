/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";

import { useTranslation } from "@plane/i18n";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
import { ProjectSettingsFeatureControlItem } from "@/components/settings/project/content/feature-control-item";
import { ProjectSettingsAccess } from "./access";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";

const featureProperties = {
  cycles: "cycle_view",
  intake: "inbox_view",
  modules: "module_view",
  pages: "page_view",
  views: "issue_views_view",
} as const;

type Props = {
  workspaceSlug: string;
  projectId: string;
  feature: keyof typeof featureProperties;
  header: ReactNode;
};

export const ProjectFeatureSettings = function ProjectFeatureSettings({
  workspaceSlug,
  projectId,
  feature,
  header,
}: Props) {
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const project = projectRows?.find((candidate) => candidate.id === projectId);
  const { t } = useTranslation();
  const pageTitle = project?.name
    ? `${project.name} settings - ${t(`project_settings.features.${feature}.short_title`)}`
    : undefined;
  return (
    <ProjectSettingsAccess workspaceSlug={workspaceSlug} projectId={projectId}>
      <SettingsContentWrapper header={header}>
        <PageHead title={pageTitle} />
        <section className="w-full">
          <SettingsHeading
            title={t(`project_settings.features.${feature}.title`)}
            description={t(`project_settings.features.${feature}.description`)}
          />
          <div className="mt-7">
            <ProjectSettingsFeatureControlItem
              title={t(`project_settings.features.${feature}.toggle_title`)}
              description={t(`project_settings.features.${feature}.toggle_description`)}
              featureProperty={featureProperties[feature]}
              projectId={projectId}
              value={!!project?.[featureProperties[feature]]}
              workspaceSlug={workspaceSlug}
            />
          </div>
        </section>
      </SettingsContentWrapper>
    </ProjectSettingsAccess>
  );
};
