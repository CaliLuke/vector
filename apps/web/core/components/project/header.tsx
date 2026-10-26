/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useParams } from "@tanstack/react-router";

// i18n
import { EUserPermissions, PROJECT_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// ui
import { Button } from "@plane/propel/button";
import { ProjectIcon } from "@plane/propel/icons";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
// plane web constants
// components
import HeaderFilters from "./filters";
import { ProjectSearch } from "./search-projects";

export function ProjectsBaseHeader() {
  // i18n
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateProjectModal } = useCommandPalette();
  const { workspaceSlug } = useParams({ strict: false });
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);

  const pathname = useLocation({ select: (location) => location.pathname });
  // auth
  const isAuthorizedUser = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  const isArchived = pathname.includes("/archives");

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs>
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={t("workspace_projects.label", { count: 2 })}
                icon={<ProjectIcon className="h-4 w-4 text-tertiary" />}
              />
            }
          />
          {isArchived && <Breadcrumbs.Item component={<BreadcrumbLink label="Archived" />} />}
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        <ProjectSearch />
        <div className="hidden md:flex">
          <HeaderFilters />
        </div>
        {isAuthorizedUser && !isArchived ? (
          <Button
            variant="primary"
            size="lg"
            onClick={() => {
              toggleCreateProjectModal(true);
            }}
            data-ph-element={PROJECT_TRACKER_ELEMENTS.CREATE_HEADER_BUTTON}
            className="items-center gap-1"
          >
            <span className="hidden sm:inline-block">{t("workspace_projects.create.label")}</span>
            <span className="inline-block sm:hidden">{t("workspace_projects.label", { count: 1 })}</span>
          </Button>
        ) : (
          <></>
        )}
      </Header.RightItem>
    </Header>
  );
}
