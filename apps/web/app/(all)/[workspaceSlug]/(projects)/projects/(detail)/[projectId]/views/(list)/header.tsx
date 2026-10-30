/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// ui
import { PROJECT_VIEW_TRACKER_ELEMENTS } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { ViewsIcon } from "@plane/propel/icons";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { ViewListHeader } from "@/components/views/view-list-header";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useRouteProject } from "@/hooks/use-projects";
// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

export const ProjectViewsHeader = function ProjectViewsHeader() {
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/views",
  });
  // store hooks
  const { toggleCreateViewModal } = useCommandPalette();
  const routeProjectQuery = useRouteProject();
  const loader = routeProjectQuery.isPending;

  return (
    <>
      <Header>
        <Header.LeftItem>
          <Breadcrumbs isLoading={loader}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label="Views"
                  href={`/${workspaceSlug}/projects/${projectId}/views/`}
                  icon={<ViewsIcon className="h-4 w-4 text-tertiary" />}
                  isLast
                />
              }
              isLast
            />
          </Breadcrumbs>
        </Header.LeftItem>
        <Header.RightItem>
          <ViewListHeader />
          <div>
            <Button
              data-ph-element={PROJECT_VIEW_TRACKER_ELEMENTS.RIGHT_HEADER_ADD_BUTTON}
              variant="primary"
              size="lg"
              onClick={() => toggleCreateViewModal(true)}
            >
              Add view
            </Button>
          </div>
        </Header.RightItem>
      </Header>
    </>
  );
};
