/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";

// plane imports
import { PageIcon } from "@plane/propel/icons";
import type { ICustomSearchSelectOption } from "@plane/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@plane/ui";
import { getPageName } from "@plane/utils/page";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { PageAccessIcon } from "@/components/common/page-access-icon";
import { SwitcherIcon, SwitcherLabel } from "@/components/common/switcher-label";
import { PageHeaderActions } from "@/components/pages/header/actions";
import { PageSyncingBadge } from "@/components/pages/header/syncing-badge";
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
// hooks
import { useRouteProject } from "@/hooks/use-projects";

import { usePageRecord } from "@/components/pages/page-record-context";

export const PageDetailsHeader = function PageDetailsHeader() {
  // router
  const router = useRouter();
  const { workspaceSlug, pageId, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/pages/$pageId",
  });
  // store hooks
  const projectQuery = useRouteProject();
  const { page, pages, pageQuery, pageListQuery, syncStatus } = usePageRecord();
  // derived values
  const switcherOptions = pages
    .filter((_page) => _page.project_ids?.includes(projectId))
    .map((_page) => {
      return {
        value: _page.id,
        query: _page.name,
        content: (
          <div className="flex items-center justify-between gap-2">
            <SwitcherLabel logo_props={_page.logo_props} name={getPageName(_page.name)} LabelIcon={PageIcon} />
            <PageAccessIcon {..._page} />
          </div>
        ),
      };
    })
    .filter((option) => option !== undefined) as ICustomSearchSelectOption[];

  if (!page) return null;

  return (
    <Header>
      <Header.LeftItem>
        <div>
          <Breadcrumbs isLoading={projectQuery.isPending || pageListQuery.isPending || pageQuery.isPending}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label="Pages"
                  href={`/${workspaceSlug}/projects/${projectId}/pages/`}
                  icon={<PageIcon className="h-4 w-4 text-tertiary" />}
                />
              }
            />

            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={pageId ?? ""}
                  navigationItems={switcherOptions}
                  onChange={(value: string) => {
                    void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/pages/${value}` });
                  }}
                  title={getPageName(page?.name)}
                  icon={
                    <Breadcrumbs.Icon>
                      <SwitcherIcon logo_props={page.logo_props} LabelIcon={PageIcon} size={16} />
                    </Breadcrumbs.Icon>
                  }
                  isLast
                />
              }
            />
          </Breadcrumbs>
        </div>
      </Header.LeftItem>
      <Header.RightItem>
        <PageSyncingBadge syncStatus={syncStatus} />
        <PageHeaderActions page={page} workspaceSlug={workspaceSlug} />
      </Header.RightItem>
    </Header>
  );
};
