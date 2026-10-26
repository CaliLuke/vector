/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter, useLocation } from "@tanstack/react-router";
import { useCallback } from "react";

// constants
import { EUserPermissions } from "@plane/constants";
// plane types
import { Button } from "@plane/propel/button";
import { PageIcon } from "@plane/propel/icons";
// plane ui
import { Breadcrumbs, Header } from "@plane/ui";
// helpers
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
// hooks
import { useRouteProject } from "@/hooks/use-projects";
// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCreatePageAction } from "@/components/pages/use-create-page-action";

export const PagesListHeader = function PagesListHeader() {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/pages",
  });
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const pageType = searchParams.get("type");
  // store hooks
  const projectQuery = useRouteProject();
  const currentProjectDetails = projectQuery.project;
  const handlePageCreated = useCallback(
    (pageId: string) => {
      const href = `/${workspaceSlug}/projects/${currentProjectDetails?.id}/pages/${pageId}`;
      void router.navigate({ href });
    },
    [currentProjectDetails?.id, router, workspaceSlug]
  );
  const { createPage: handleCreatePage, isCreatingPage } = useCreatePageAction({
    workspaceSlug,
    projectId,
    pageType: pageType === "private" ? "private" : "public",
    onCreated: handlePageCreated,
  });
  const { can } = useProjectPermissions(workspaceSlug, projectId);
  const canCurrentUserCreatePage = can([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs isLoading={projectQuery.isPending}>
          <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug?.toString()} projectId={projectId?.toString()} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label="Pages"
                href={`/${workspaceSlug}/projects/${currentProjectDetails?.id}/pages/`}
                icon={<PageIcon className="h-4 w-4 text-tertiary" />}
                isLast
              />
            }
            isLast
          />
        </Breadcrumbs>
      </Header.LeftItem>
      {canCurrentUserCreatePage && (
        <Header.RightItem>
          <Button variant="primary" size="lg" onClick={() => void handleCreatePage()} loading={isCreatingPage}>
            {isCreatingPage ? "Adding" : "Add page"}
          </Button>
        </Header.RightItem>
      )}
    </Header>
  );
};
