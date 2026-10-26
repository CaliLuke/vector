/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";

// plane imports
import { cn } from "@plane/utils/common";
// assets
import emptyCycle from "@/app/assets/empty-state/cycle.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
import { CycleDetailsSidebar } from "@/components/cycles/analytics-sidebar";
import { CycleLayoutRoot } from "@/components/issues/issue-layouts/roots/cycle-layout-root";
// hooks
import { useParentRemovalNavigation } from "@/hooks/use-parent-removal-navigation";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useProjectDetail } from "@/hooks/use-projects";

import { useParentChartPreferences } from "@/providers/cycle-module-preferences";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";

function CycleDetailPage(): React.JSX.Element | null {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId, cycleId } = Route.useParams();
  // store hooks
  const detail = useCycleDetails(workspaceSlug, projectId, cycleId);
  useParentRemovalNavigation(
    JSON.stringify([detail.scope, cycleId]),
    detail.cycle === null,
    `/${workspaceSlug}/projects/${projectId}/cycles`
  );
  const { project } = useProjectDetail(workspaceSlug, projectId);
  // hooks
  const chart = useParentChartPreferences("cycle", workspaceSlug, projectId, cycleId);
  const isSidebarCollapsed = chart.sidebarCollapsed;
  const cycle = detail.cycle;
  const pageTitle = project?.name && cycle?.name ? `${project?.name} - ${cycle?.name}` : undefined;

  /**
   * Toggles the sidebar
   */
  const toggleSidebar = () => chart.update({ sidebarCollapsed: !isSidebarCollapsed });

  // const activeLayout = issuesFilter?.issueFilters?.displayFilters?.layout;
  return (
    <>
      <PageHead title={pageTitle} />
      {detail.isError || detail.isAccessDenied ? (
        <CatalogueReadFailure label="cycles" retry={detail.retry} />
      ) : detail.query.isSuccess && !cycle ? (
        <EmptyState
          image={emptyCycle}
          title="Cycle does not exist"
          description="The cycle you are looking for does not exist or has been deleted."
          primaryButton={{
            text: "View other cycles",
            onClick: () => void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/cycles` }),
          }}
        />
      ) : (
        <>
          <div className="flex h-full w-full">
            <div className="h-full w-full overflow-hidden">
              <CycleLayoutRoot />
            </div>
            {!isSidebarCollapsed && (
              <div
                className={cn(
                  "vertical-scrollbar absolute right-0 z-13 flex scrollbar-sm h-full w-[21.5rem] flex-shrink-0 flex-col gap-3.5 overflow-y-auto border-l border-subtle bg-surface-1 px-4 shadow-raised-200 duration-300"
                )}
              >
                <CycleDetailsSidebar
                  handleClose={toggleSidebar}
                  cycleId={cycleId}
                  projectId={projectId}
                  workspaceSlug={workspaceSlug}
                />
              </div>
            )}
          </div>
        </>
      )}
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/cycles/$cycleId/")({
  component: CycleDetailPage,
});
