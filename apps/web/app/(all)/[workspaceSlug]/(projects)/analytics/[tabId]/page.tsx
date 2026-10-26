/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";

// plane package imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import { Tabs } from "@plane/propel/tabs";
// components
import { cn } from "@plane/utils/common";
import AnalyticsFilterActions from "@/components/analytics/analytics-filter-actions";
import { PageHead } from "@/components/core/page-title";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useAnalyticsTabs } from "@/components/analytics/use-analytics-tabs";
import { AnalyticsStateProvider } from "@/providers/analytics-state-provider";

function AnalyticsPage(): React.JSX.Element | null {
  const params = Route.useParams();
  const { tabId } = params;

  // hooks
  const router = useRouter();

  // plane imports
  const { t } = useTranslation();

  // store hooks
  const { toggleCreateProjectModal } = useCommandPalette();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const loader = projectCatalogue.isPending;
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { can: canWorkspace } = useWorkspacePermissions(params.workspaceSlug);

  const workspaceProjectIds = projectRows
    ?.filter((project) => project.workspace === currentWorkspace?.id && !project.archived_at)
    .map((project) => project.id);

  const pageTitle = currentWorkspace?.name
    ? t(`workspace_analytics.page_label`, { workspace: currentWorkspace?.name })
    : undefined;

  // permissions
  const canPerformEmptyStateActions = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const workspaceSlug = params.workspaceSlug;
  const ANALYTICS_TABS = useAnalyticsTabs(workspaceSlug.toString());

  const handleTabChange = (value: string) => {
    void router.navigate({ href: `/${workspaceSlug}/analytics/${value}` });
  };

  return (
    <AnalyticsStateProvider workspaceSlug={workspaceSlug}>
      <>
        <PageHead title={pageTitle} />
        {workspaceProjectIds && (
          <>
            {workspaceProjectIds.length > 0 || loader ? (
              <div className="flex h-full overflow-hidden">
                <Tabs value={tabId || ANALYTICS_TABS[0]?.key} onValueChange={handleTabChange} className="h-full w-full">
                  <div className={"flex h-full w-full flex-col"}>
                    <div
                      className={cn(
                        "flex w-full items-center justify-between gap-4 overflow-hidden border-b border-subtle bg-surface-1 px-6 py-2"
                      )}
                    >
                      <Tabs.List className={"flex h-7 w-fit overflow-x-auto"}>
                        {ANALYTICS_TABS.map((tab) => (
                          <Tabs.Trigger
                            key={tab.key}
                            value={tab.key}
                            disabled={tab.isDisabled}
                            size="md"
                            className="h-6 px-3"
                          >
                            {tab.label}
                          </Tabs.Trigger>
                        ))}
                      </Tabs.List>

                      <div className="flex-shrink-0">
                        <AnalyticsFilterActions />
                      </div>
                    </div>
                    {ANALYTICS_TABS.map((tab) => (
                      <Tabs.Content
                        key={tab.key}
                        value={tab.key}
                        className={"h-full overflow-hidden overflow-y-auto px-2"}
                      >
                        <tab.content />
                      </Tabs.Content>
                    ))}
                  </div>
                </Tabs>
              </div>
            ) : (
              <EmptyStateDetailed
                assetKey="project"
                title={t("workspace_projects.empty_state.no_projects.title")}
                description={t("workspace_projects.empty_state.no_projects.description")}
                actions={[
                  {
                    label: "Create a project",
                    onClick: () => {
                      toggleCreateProjectModal(true);
                    },
                    disabled: !canPerformEmptyStateActions,
                  },
                ]}
              />
            )}
          </>
        )}
      </>
    </AnalyticsStateProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/analytics/$tabId/")({
  component: AnalyticsPage,
});
