/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";

import { ArchiveIcon, CycleIcon, ModuleIcon, WorkItemsIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { EIssuesStoreType } from "@plane/types";
// ui
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
// hooks
import { useRouteProject } from "@/hooks/use-projects";
import { useIssueListCount } from "@/hooks/use-issue-list-count";

import { usePlatformOS } from "@/hooks/use-platform-os";
// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

type TProps = {
  activeTab: "issues" | "cycles" | "modules";
};

const PROJECT_ARCHIVES_BREADCRUMB_LIST: {
  [key: string]: {
    label: string;
    href: string;
    icon: React.FC<React.SVGAttributes<SVGElement> & { className?: string }>;
  };
} = {
  issues: {
    label: "Work items",
    href: "/issues",
    icon: WorkItemsIcon,
  },
  cycles: {
    label: "Cycles",
    href: "/cycles",
    icon: CycleIcon,
  },
  modules: {
    label: "Modules",
    href: "/modules",
    icon: ModuleIcon,
  },
};

export function ProjectArchivesHeader(props: TProps) {
  const { activeTab } = props;
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // store hooks
  const { count: issueCount } = useIssueListCount(EIssuesStoreType.ARCHIVED);
  const routeProjectQuery = useRouteProject();
  const loader = routeProjectQuery.isPending;
  // hooks
  const { isMobile } = usePlatformOS();

  const activeTabBreadcrumbDetail =
    PROJECT_ARCHIVES_BREADCRUMB_LIST[activeTab as keyof typeof PROJECT_ARCHIVES_BREADCRUMB_LIST];

  if (!workspaceSlug || !projectId) return null;

  return (
    <Header>
      <Header.LeftItem>
        <div className="flex items-center gap-2.5">
          <Breadcrumbs onBack={router.history.back} isLoading={loader}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  href={`/${workspaceSlug}/projects/${projectId}/archives/issues`}
                  label="Archives"
                  icon={<ArchiveIcon className="h-4 w-4 text-tertiary" />}
                />
              }
            />
            {activeTabBreadcrumbDetail && (
              <Breadcrumbs.Item
                component={
                  <BreadcrumbLink
                    label={activeTabBreadcrumbDetail.label}
                    icon={<activeTabBreadcrumbDetail.icon className="h-4 w-4 text-tertiary" />}
                  />
                }
              />
            )}
          </Breadcrumbs>
          {activeTab === "issues" && issueCount && issueCount > 0 ? (
            <Tooltip
              isMobile={isMobile}
              tooltipContent={`There are ${issueCount} ${issueCount > 1 ? "work items" : "work item"} in project's archived`}
              position="bottom"
            >
              <span className="flex flex-shrink-0 cursor-default items-center justify-center rounded-xl bg-accent-primary/20 px-2.5 py-0.5 text-center text-11 font-semibold text-accent-primary">
                {issueCount}
              </span>
            </Tooltip>
          ) : null}
        </div>
      </Header.LeftItem>
    </Header>
  );
}
