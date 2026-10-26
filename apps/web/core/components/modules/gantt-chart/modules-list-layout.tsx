/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// PLane
import { EUserPermissions, GANTT_TIMELINE_TYPE } from "@plane/types";
import type { IBlockUpdateData, IBlockUpdateDependencyData, IModule } from "@plane/types";
// components
import { GanttChartRoot, ModuleGanttSidebar } from "@/components/gantt-chart";
import { TimeLineTypeContext } from "@/components/gantt-chart/contexts";
import { ModuleGanttBlock } from "@/components/modules";
// hooks
import { useProjectModules } from "@/hooks/use-modules";
import { useFavorites } from "@/hooks/use-favorites";
import { visibleModules } from "@/queries/cycle-module-selectors";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useModuleCataloguePreferences } from "@/providers/cycle-module-preferences";
import { useProjectPermissions } from "@/hooks/use-permissions";

const renderModuleSidebar = (props: Parameters<typeof ModuleGanttSidebar>[0]) => <ModuleGanttSidebar {...props} />;
const renderModuleBlock = (data: IModule) => <ModuleGanttBlock module={data} />;

export const ModulesListGanttChartView = function ModulesListGanttChartView() {
  // router
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId",
  });
  // store
  const { can } = useProjectPermissions(workspaceSlug, projectId);
  const catalogue = useProjectModules(workspaceSlug, projectId);
  const favorites = useFavorites(workspaceSlug);
  const preferences = useModuleCataloguePreferences(workspaceSlug, projectId);
  const { display: displayFilters } = preferences;

  // derived values
  const rows = visibleModules(
    catalogue.modules,
    preferences.display,
    preferences.filters,
    preferences.search,
    new Set(
      favorites.favorites?.flatMap((row) =>
        row.entity_type === "module" && row.entity_identifier ? [row.entity_identifier] : []
      ) ?? []
    )
  );
  const filteredModuleIds = rows?.map((row) => row.id);

  const handleModuleUpdate = async (module: IModule, data: IBlockUpdateData) => {
    if (!workspaceSlug || !module) return;

    const payload: Partial<IModule> = {
      ...(data.start_date !== undefined ? { start_date: data.start_date } : {}),
      ...(data.target_date !== undefined ? { target_date: data.target_date } : {}),
      ...(data.sort_order ? { sort_order: data.sort_order.newSortOrder } : {}),
    };

    await catalogue.mutation.mutateAsync({ type: "patch", id: module.id, data: payload });
  };

  const updateBlockDates = async (blockUpdates: IBlockUpdateDependencyData[]) => {
    const blockUpdate = blockUpdates[0];

    if (!blockUpdate) return;

    const payload: Partial<IModule> = {};

    if (blockUpdate.start_date) payload.start_date = blockUpdate.start_date;
    if (blockUpdate.target_date) payload.target_date = blockUpdate.target_date;

    await catalogue.mutation.mutateAsync({ type: "patch", id: blockUpdate.id, data: payload });
  };

  const isAllowed = can([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="modules" retry={catalogue.retry} />;
  if (!filteredModuleIds || (displayFilters.favorites && favorites.favorites === undefined))
    return <output>Loading modules…</output>;

  return (
    <TimeLineTypeContext.Provider value={GANTT_TIMELINE_TYPE.MODULE}>
      <GanttChartRoot
        title="Modules"
        loaderTitle="Modules"
        blockIds={filteredModuleIds}
        blockItems={rows ?? []}
        sidebarToRender={renderModuleSidebar}
        blockUpdateHandler={(block, payload) => handleModuleUpdate(block, payload)}
        blockToRender={renderModuleBlock}
        enableBlockLeftResize={isAllowed}
        enableBlockRightResize={isAllowed}
        enableBlockMove={isAllowed}
        enableReorder={isAllowed && displayFilters?.order_by === "sort_order"}
        enableAddBlock={isAllowed}
        updateBlockDates={updateBlockDates}
        showAllBlocks
      />
    </TimeLineTypeContext.Provider>
  );
};
