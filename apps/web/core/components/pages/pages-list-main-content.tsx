/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { useCallback } from "react";
// plane imports

import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import type { TPageNavigationTabs } from "@plane/types";
import { filterPagesByPageType } from "@plane/utils/page";
// components
import { PageLoader } from "@/components/pages/loaders/page-loader";
import { useProjectPermissions } from "@/hooks/use-permissions";
// plane web hooks
import { usePageList } from "./page-record-context";
import { useCreatePageAction } from "./use-create-page-action";

type Props = {
  children: React.ReactNode;
  pageType: TPageNavigationTabs;
};

export function PagesListMainContent(props: Props) {
  const { children, pageType } = props;
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/pages",
  });
  const { pages, visiblePages, isPending, isError, isDenied, refetch } = usePageList();
  // router
  const router = useRouter();
  const handlePageCreated = useCallback(
    (pageId: string) => {
      void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/pages/${pageId}` });
    },
    [projectId, router, workspaceSlug]
  );
  const { createPage: handleCreatePage, isCreatingPage } = useCreatePageAction({
    workspaceSlug,
    projectId,
    pageType,
    onCreated: handlePageCreated,
  });
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const canPerformEmptyStateActions = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  if (isPending && pages.length === 0 && !isDenied && !isError) return <PageLoader />;
  if (isDenied)
    return (
      <EmptyStateDetailed
        assetKey="page"
        title="Pages are unavailable"
        description="You do not have access to the pages in this project."
      />
    );
  if (isError && pages.length === 0)
    return (
      <div className="grid h-full place-items-center">
        <div className="text-center">
          <p className="text-13 text-secondary">Failed to load pages.</p>
          <Button variant="secondary" size="sm" onClick={() => void refetch()} className="mt-3">
            Retry
          </Button>
        </div>
      </div>
    );
  const pageIds = filterPagesByPageType(pageType, pages).filter((page) => page.project_ids?.includes(projectId));
  // if no pages exist in the active page type
  if (pageIds.length === 0) {
    if (pageType === "public" || pageType === "private") {
      return (
        <EmptyStateDetailed
          assetKey="page"
          title={t("project_empty_state.pages.title")}
          description={t("project_empty_state.pages.description")}
          actions={[
            {
              label: t("project_empty_state.pages.cta_primary"),
              onClick: () => {
                void handleCreatePage();
              },
              variant: "primary",
              disabled: !canPerformEmptyStateActions || isCreatingPage,
            },
          ]}
        />
      );
    }
    if (pageType === "archived")
      return (
        <EmptyStateDetailed
          assetKey="page"
          title={t("project_empty_state.archive_pages.title")}
          description={t("project_empty_state.archive_pages.description")}
        />
      );
  }
  // if no pages match the filter criteria
  if (visiblePages.length === 0)
    return (
      <EmptyStateDetailed
        assetKey="search"
        title={t("common_empty_state.search.title")}
        description={t("common_empty_state.search.description")}
      />
    );

  return <div className="h-full w-full overflow-hidden">{children}</div>;
}
