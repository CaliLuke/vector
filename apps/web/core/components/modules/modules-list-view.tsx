/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation } from "@tanstack/react-router";

// components
import { MODULE_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import type { IModule } from "@plane/types";
import { EUserProjectRoles } from "@plane/types";
import { ContentWrapper, Row, ERowVariant } from "@plane/ui";
// components
import { ListLayout } from "@/components/core/list";
import { ModuleCardItem, ModuleListItem, ModulePeekOverview, ModulesListGanttChartView } from "@/components/modules";
import { CycleModuleBoardLayoutLoader } from "@/components/ui/loader/cycle-module-board-loader";
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
import { GanttLayoutLoader } from "@/components/ui/loader/layouts/gantt-layout-loader";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useProjectModules } from "@/hooks/use-modules";
import { useFavorites } from "@/hooks/use-favorites";
import { favoriteModuleIds, visibleModules } from "@/queries/cycle-module-selectors";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useModuleCataloguePreferences } from "@/providers/cycle-module-preferences";
import { useProjectPermissions } from "@/hooks/use-permissions";

type ModuleLayoutRowsProps = {
  modules: readonly IModule[];
  layout: string | undefined;
  peekModule: string | null;
};

function ModuleLayoutRows(props: ModuleLayoutRowsProps) {
  const { modules, layout, peekModule } = props;
  return (
    <>
      {layout === "list" && (
        <ListLayout>
          {modules.map((module) => (
            <ModuleListItem key={module.id} module={module} />
          ))}
        </ListLayout>
      )}
      {layout === "board" && (
        <Row
          className={`grid size-full grid-cols-1 gap-6 overflow-y-auto py-page-y ${
            peekModule
              ? "3xl:grid-cols-3 lg:grid-cols-1 xl:grid-cols-2"
              : "3xl:grid-cols-4 lg:grid-cols-2 xl:grid-cols-3"
          } vertical-scrollbar scrollbar-lg auto-rows-max transition-all`}
        >
          {modules.map((module) => (
            <ModuleCardItem key={module.id} module={module} />
          ))}
        </Row>
      )}
      {layout === "gantt" && (
        <div className="size-full overflow-hidden">
          <ModulesListGanttChartView />
        </div>
      )}
    </>
  );
}

function ModuleLayoutLoader(props: { layout: string | undefined }) {
  const { layout } = props;
  return (
    <>
      {layout === "list" && <CycleModuleListLayoutLoader />}
      {layout === "board" && <CycleModuleBoardLayoutLoader />}
      {layout === "gantt" && <GanttLayoutLoader />}
    </>
  );
}

function ModuleEmptyState(props: {
  title: string;
  description: string;
  createLabel?: string;
  onCreate?: () => void;
  canCreate?: boolean;
}) {
  const { title, description, createLabel, onCreate, canCreate = false } = props;
  if (!onCreate) return <EmptyStateDetailed assetKey="search" title={title} description={description} />;
  return (
    <EmptyStateDetailed
      assetKey="module"
      title={title}
      description={description}
      actions={[
        {
          label: createLabel ?? "Create module",
          onClick: onCreate,
          disabled: !canCreate,
          variant: "primary",
          "data-ph-element": MODULE_TRACKER_ELEMENTS.EMPTY_STATE_ADD_BUTTON,
        },
      ]}
    />
  );
}

function isModuleCataloguePending(props: {
  needsFavorites: boolean;
  favorites: ReturnType<typeof useFavorites>["favorites"];
  isPending: boolean;
  modules: ReturnType<typeof useProjectModules>["modules"];
  visibleModules: ReturnType<typeof visibleModules>;
}) {
  return (props.needsFavorites && !props.favorites) || props.isPending || !props.modules || !props.visibleModules;
}

function getModuleCataloguePlaceholder(props: {
  favoritesEnabled: boolean;
  favoritesError: boolean;
  retryFavorites: () => Promise<unknown>;
  catalogueError: boolean;
  retryCatalogue: () => Promise<unknown>;
  isPending: boolean;
  favorites: ReturnType<typeof useFavorites>["favorites"];
  catalogueModules: ReturnType<typeof useProjectModules>["modules"];
  visibleModules: ReturnType<typeof visibleModules>;
  layout: string | undefined;
  translate: ReturnType<typeof useTranslation>["t"];
  onCreate: () => void;
  canCreate: boolean;
}): React.ReactNode {
  if (props.favoritesEnabled && props.favoritesError)
    return <CatalogueReadFailure label="favorites" retry={props.retryFavorites} />;
  if (props.catalogueError) return <CatalogueReadFailure label="modules" retry={props.retryCatalogue} />;
  if (
    isModuleCataloguePending({
      needsFavorites: props.favoritesEnabled,
      favorites: props.favorites,
      isPending: props.isPending,
      modules: props.catalogueModules,
      visibleModules: props.visibleModules,
    })
  )
    return <ModuleLayoutLoader layout={props.layout} />;
  if (!props.catalogueModules || !props.visibleModules) return <ModuleLayoutLoader layout={props.layout} />;
  if (props.catalogueModules.length === 0)
    return (
      <ModuleEmptyState
        title={props.translate("project_empty_state.modules.title")}
        description={props.translate("project_empty_state.modules.description")}
        createLabel={props.translate("project_empty_state.modules.cta_primary")}
        onCreate={props.onCreate}
        canCreate={props.canCreate}
      />
    );
  if (props.visibleModules.length === 0)
    return (
      <ModuleEmptyState
        title={props.translate("common_empty_state.search.title")}
        description={props.translate("common_empty_state.search.description")}
      />
    );
  return null;
}

export const ModulesListView = function ModulesListView() {
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const peekModule = searchParams.get("peekModule");
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateModuleModal } = useCommandPalette();
  const catalogue = useProjectModules(workspaceSlug, projectId, false);
  const favorites = useFavorites(workspaceSlug);
  const preferences = useModuleCataloguePreferences(workspaceSlug, projectId);
  const { display: displayFilters } = preferences;
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // derived values

  const favoriteIds = favoriteModuleIds(favorites.favorites);
  const modules = visibleModules(
    catalogue.modules,
    preferences.display,
    preferences.filters,
    preferences.search,
    favoriteIds
  );
  const canPerformEmptyStateActions = canProject([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]);

  const placeholder = getModuleCataloguePlaceholder({
    favoritesEnabled: Boolean(preferences.display.favorites),
    favoritesError: favorites.isError,
    retryFavorites: favorites.retry,
    catalogueError: catalogue.isError || catalogue.isAccessDenied,
    retryCatalogue: catalogue.retry,
    isPending: catalogue.isPending,
    favorites: favorites.favorites,
    catalogueModules: catalogue.modules,
    visibleModules: modules,
    layout: displayFilters?.layout,
    translate: t,
    onCreate: () => toggleCreateModuleModal(true),
    canCreate: canPerformEmptyStateActions ?? false,
  });
  if (placeholder) return placeholder;
  if (!catalogue.modules || !modules) return <ModuleLayoutLoader layout={displayFilters?.layout} />;

  return (
    <ContentWrapper variant={ERowVariant.HUGGING}>
      <div className="flex size-full justify-between">
        <ModuleLayoutRows modules={modules} layout={displayFilters?.layout} peekModule={peekModule} />
        <div className="flex-shrink-0">
          <ModulePeekOverview projectId={projectId?.toString() ?? ""} workspaceSlug={workspaceSlug?.toString() ?? ""} />
        </div>
      </div>
    </ContentWrapper>
  );
};
