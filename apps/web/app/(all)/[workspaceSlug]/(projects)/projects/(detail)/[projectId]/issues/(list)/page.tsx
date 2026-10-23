/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { observer } from "mobx-react";
// i18n
import { useTranslation } from "@plane/i18n";
// components
import { PageHead } from "@/components/core/page-title";
import { ProjectLayoutRoot } from "@/components/issues/issue-layouts/roots/project-layout-root";
// hooks
import { useProject } from "@/hooks/store/use-project";

function ProjectIssuesPage(): React.JSX.Element | null {
  const { projectId } = Route.useParams();
  // i18n
  const { t } = useTranslation();
  // store
  const { getProjectById } = useProject();

  // derived values
  const project = getProjectById(projectId);
  const pageTitle = project?.name ? `${project?.name} - ${t("issue.label", { count: 2 })}` : undefined; // Count is for pluralization

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="h-full w-full">
        <ProjectLayoutRoot />
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/issues/")({
  component: observer(ProjectIssuesPage),
});
