/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";

// ui
import { EUserPermissions, CYCLE_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { CycleIcon } from "@plane/propel/icons";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { CyclesViewHeader } from "@/components/cycles/cycles-view-header";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";

// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

export const CyclesListHeader = function CyclesListHeader() {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/cycles",
  });

  // store hooks
  const { toggleCreateCycleModal } = useCommandPalette();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const loader = routeProjectQuery.isPending;
  const { t } = useTranslation();

  const canUserCreateCycle = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs onBack={router.history.back} isLoading={loader}>
          <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug?.toString()} projectId={projectId?.toString()} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label="Cycles"
                href={`/${workspaceSlug}/projects/${currentProjectDetails?.id}/cycles/`}
                icon={<CycleIcon className="h-4 w-4 text-tertiary" />}
                isLast
              />
            }
            isLast
          />
        </Breadcrumbs>
      </Header.LeftItem>
      {canUserCreateCycle && currentProjectDetails ? (
        <Header.RightItem>
          <CyclesViewHeader projectId={currentProjectDetails.id} />
          <Button
            variant="primary"
            size="lg"
            data-ph-element={CYCLE_TRACKER_ELEMENTS.RIGHT_HEADER_ADD_BUTTON}
            onClick={() => {
              toggleCreateCycleModal(true);
            }}
          >
            <div className="block sm:hidden">{t("add")}</div>
            <div className="hidden sm:block">{t("project_cycles.add_cycle")}</div>
          </Button>
        </Header.RightItem>
      ) : (
        <></>
      )}
    </Header>
  );
};
