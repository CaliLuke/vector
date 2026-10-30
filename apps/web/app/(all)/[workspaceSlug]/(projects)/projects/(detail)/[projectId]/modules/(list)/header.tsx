/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";

// plane imports
import { EUserPermissions, MODULE_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// ui
import { Button } from "@plane/propel/button";
import { ModuleIcon } from "@plane/propel/icons";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { ModuleViewHeader } from "@/components/modules";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";

// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

export const ModulesListHeader = function ModulesListHeader() {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/modules",
  });
  // store hooks
  const { toggleCreateModuleModal } = useCommandPalette();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const routeProjectQuery = useRouteProject();
  const loader = routeProjectQuery.isPending;

  const { t } = useTranslation();

  // auth
  const canUserCreateModule = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <Header>
      <Header.LeftItem>
        <div>
          <Breadcrumbs onBack={router.history.back} isLoading={loader}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label="Modules"
                  href={`/${workspaceSlug}/projects/${projectId}/modules/`}
                  icon={<ModuleIcon className="h-4 w-4 text-tertiary" />}
                  isLast
                />
              }
              isLast
            />
          </Breadcrumbs>
        </div>
      </Header.LeftItem>
      <Header.RightItem>
        <ModuleViewHeader />
        {canUserCreateModule ? (
          <Button
            variant="primary"
            data-ph-element={MODULE_TRACKER_ELEMENTS.RIGHT_HEADER_ADD_BUTTON}
            onClick={() => {
              toggleCreateModuleModal(true);
            }}
            size="lg"
          >
            <div className="block sm:hidden">{t("add")}</div>
            <div className="hidden sm:block">{t("project_module.add_module")}</div>
          </Button>
        ) : (
          <></>
        )}
      </Header.RightItem>
    </Header>
  );
};
