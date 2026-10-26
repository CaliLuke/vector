/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";
import { useState } from "react";

// plane imports
import { useTheme } from "next-themes";
import { CYCLE_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import type { TCycleFilters } from "@plane/types";
import { EUserProjectRoles } from "@plane/types";
// components
import { Header, EHeaderVariant } from "@plane/ui";
import { calculateTotalFilters } from "@plane/utils/filter";
// assets
import darkEmptyState from "@/app/assets/empty-state/disabled-feature/cycles-dark.webp?url";
import lightEmptyState from "@/app/assets/empty-state/disabled-feature/cycles-light.webp?url";
// components
import { PageHead } from "@/components/core/page-title";
import { CycleAppliedFiltersList } from "@/components/cycles/applied-filters";
import { CyclesView } from "@/components/cycles/cycles-view";
import { CycleCreateUpdateModal } from "@/components/cycles/modal";
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useCycleCataloguePreferences } from "@/providers/cycle-module-preferences";
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";

function ProjectCyclesPage(): React.JSX.Element | null {
  // states
  const [createModal, setCreateModal] = useState(false);
  // store hooks
  const { project: currentProjectDetails } = useRouteProject();
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = Route.useParams();
  const catalogue = useProjectCycles(workspaceSlug, projectId);
  // theme hook
  const { resolvedTheme } = useTheme();
  // plane hooks
  const { t } = useTranslation();
  // cycle filters hook
  const preferences = useCycleCataloguePreferences(workspaceSlug, projectId);
  const { filters: currentProjectFilters } = preferences;
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const resolvedEmptyState = resolvedTheme === "light" ? lightEmptyState : darkEmptyState;
  const totalCycles = catalogue.cycles?.length ?? 0;
  const project = currentProjectDetails;
  const pageTitle = project?.name ? `${project.name} - ${t("common.cycles", { count: 2 })}` : undefined;
  const hasAdminLevelPermission = canProject([EUserProjectRoles.ADMIN]);
  const hasMemberLevelPermission = canProject([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]);

  const handleRemoveFilter = (key: keyof TCycleFilters, value: string | null) => {
    let newValues = currentProjectFilters?.[key] ?? [];

    if (!value) newValues = [];
    else newValues = newValues.filter((val) => val !== value);

    preferences.update({ filters: { ...preferences.filters, [key]: newValues } });
  };

  // No access to cycle
  if (currentProjectDetails?.cycle_view === false)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <DetailedEmptyState
          title={t("disabled_project.empty_state.cycle.title")}
          description={t("disabled_project.empty_state.cycle.description")}
          assetPath={resolvedEmptyState}
          primaryButton={{
            text: t("disabled_project.empty_state.cycle.primary_button.text"),
            onClick: () => {
              void router.navigate({ href: `/${workspaceSlug}/settings/projects/${projectId}/features` });
            },
            disabled: !hasAdminLevelPermission,
          }}
        />
      </div>
    );

  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="cycles" retry={catalogue.retry} />;
  if (catalogue.isPending) return <CycleModuleListLayoutLoader />;

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="h-full w-full">
        <CycleCreateUpdateModal
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          isOpen={createModal}
          handleClose={() => setCreateModal(false)}
        />
        {totalCycles === 0 ? (
          <div className="h-full place-items-center">
            <EmptyStateDetailed
              assetKey="cycle"
              title={t("project_empty_state.cycles.title")}
              description={t("project_empty_state.cycles.description")}
              actions={[
                {
                  label: t("project_empty_state.cycles.cta_primary"),
                  onClick: () => setCreateModal(true),
                  variant: "primary",
                  disabled: !hasMemberLevelPermission,
                  "data-ph-element": CYCLE_TRACKER_ELEMENTS.EMPTY_STATE_ADD_BUTTON,
                },
              ]}
            />
          </div>
        ) : (
          <>
            {calculateTotalFilters(currentProjectFilters ?? {}) !== 0 && (
              <Header variant={EHeaderVariant.TERNARY}>
                <CycleAppliedFiltersList
                  appliedFilters={currentProjectFilters ?? {}}
                  handleClearAllFilters={() => preferences.update({ filters: {} })}
                  handleRemoveFilter={handleRemoveFilter}
                />
              </Header>
            )}

            <CyclesView workspaceSlug={workspaceSlug} projectId={projectId} />
          </>
        )}
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/cycles/")({
  component: ProjectCyclesPage,
});
