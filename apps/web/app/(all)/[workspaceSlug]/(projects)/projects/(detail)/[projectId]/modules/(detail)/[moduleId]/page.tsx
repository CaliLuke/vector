/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";

// plane imports
import { cn } from "@plane/utils/common";
// assets
import emptyModule from "@/app/assets/empty-state/module.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
import { ModuleLayoutRoot } from "@/components/issues/issue-layouts/roots/module-layout-root";
import { ModuleAnalyticsSidebar } from "@/components/modules";
// hooks
import { useParentRemovalNavigation } from "@/hooks/use-parent-removal-navigation";
import { useModuleDetails } from "@/hooks/use-modules";
import { useProjectDetail } from "@/hooks/use-projects";

import { useParentChartPreferences } from "@/providers/cycle-module-preferences";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

function ModuleIssuesPage(): React.JSX.Element | null {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId, moduleId } = Route.useParams();
  // store hooks
  const detail = useModuleDetails(workspaceSlug, projectId, moduleId);
  useParentRemovalNavigation(
    JSON.stringify([detail.scope, moduleId]),
    detail.module === null,
    `/${workspaceSlug}/projects/${projectId}/modules`
  );
  const { project } = useProjectDetail(workspaceSlug, projectId);
  // local storage
  const chart = useParentChartPreferences("module", workspaceSlug, projectId, moduleId);
  const isSidebarCollapsed = chart.sidebarCollapsed;
  const projectModule = detail.module;
  const pageTitle = project?.name && projectModule?.name ? `${project?.name} - ${projectModule?.name}` : undefined;

  const toggleSidebar = () => {
    chart.update({ sidebarCollapsed: !isSidebarCollapsed });
  };

  // const activeLayout = issuesFilter?.issueFilters?.displayFilters?.layout;
  return (
    <>
      <PageHead title={pageTitle} />
      {detail.isError || detail.isAccessDenied ? (
        <CatalogueReadFailure label="modules" retry={detail.retry} />
      ) : detail.query.isSuccess && !projectModule ? (
        <EmptyState
          image={emptyModule}
          title="Module does not exist"
          description="The module you are looking for does not exist or has been deleted."
          primaryButton={{
            text: "View other modules",
            onClick: () => void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/modules` }),
          }}
        />
      ) : (
        <div className="flex h-full w-full">
          <div className="h-full w-full overflow-hidden">
            <ModuleLayoutRoot />
          </div>
          {!isSidebarCollapsed && (
            <div
              className={cn(
                "vertical-scrollbar absolute right-0 z-13 flex scrollbar-sm h-full w-[24rem] flex-shrink-0 flex-col gap-3.5 overflow-y-auto border-l border-subtle bg-surface-1 px-6 shadow-raised-200 duration-300"
              )}
            >
              <ModuleAnalyticsSidebar
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                moduleId={moduleId}
                handleClose={toggleSidebar}
              />
            </div>
          )}
        </div>
      )}
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/modules/$moduleId/")(
  {
    component: ModuleIssuesPage,
  }
);
