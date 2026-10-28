/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import orderBy from "lodash-es/orderBy.js";
import { useParams, useLocation } from "@tanstack/react-router";

import { useTheme } from "next-themes";
// plane imports
import { useTranslation } from "@plane/i18n";
import type { THomeWidgetKeys, THomeWidgetProps } from "@plane/types";
// assets
import darkWidgetsAsset from "@/app/assets/empty-state/dashboard/widgets-dark.webp?url";
import lightWidgetsAsset from "@/app/assets/empty-state/dashboard/widgets-light.webp?url";
// components
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";
// hooks

import { homeWidgetsOptions } from "@/queries/home";
import { useHomeSettings } from "./provider";
import { useRouteProject } from "@/hooks/use-projects";
// local imports
import { StickiesWidget } from "../stickies/widget";
import { HomeLoader, NoProjectsEmptyState, RecentActivityWidget } from "./widgets";
import { DashboardQuickLinks } from "./widgets/links";
import { ManageWidgetsModal } from "./widgets/manage";

export const HOME_WIDGETS_LIST: {
  [key in THomeWidgetKeys]: {
    component: React.FC<THomeWidgetProps> | null;
    fullWidth: boolean;
    title: string;
  };
} = {
  quick_links: {
    component: DashboardQuickLinks,
    fullWidth: false,
    title: "home.quick_links.title_plural",
  },
  recents: {
    component: RecentActivityWidget,
    fullWidth: false,
    title: "home.recents.title",
  },
  my_stickies: {
    component: StickiesWidget,
    fullWidth: false,
    title: "stickies.title",
  },
  new_at_plane: {
    component: null,
    fullWidth: false,
    title: "home.new_at_plane.title",
  },
  quick_tutorial: {
    component: null,
    fullWidth: false,
    title: "home.quick_tutorial.title",
  },
};

export const DashboardWidgets = function DashboardWidgets() {
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // navigation
  const pathname = useLocation({ select: (location) => location.pathname });
  // theme hook
  const { resolvedTheme } = useTheme();
  // store hooks
  const { showWidgetSettings, setShowWidgetSettings } = useHomeSettings();
  const widgetsQuery = useRecordCollection(homeWidgetsOptions(workspaceSlug), (widget) => widget.key);
  const widgets = orderBy(widgetsQuery.data ?? [], "sort_order", "desc");
  const isAnyWidgetEnabled = widgets.some((widget) => widget.is_enabled);
  const loading = widgetsQuery.isPending;
  const routeProjectQuery = useRouteProject();
  const loader = routeProjectQuery.isPending;
  // plane hooks
  const { t } = useTranslation();
  // derived values
  const noWidgetsResolvedPath = resolvedTheme === "light" ? lightWidgetsAsset : darkWidgetsAsset;

  // derived values
  const isWikiApp = pathname.includes(`/${workspaceSlug.toString()}/pages`);
  if (!workspaceSlug) return null;
  if (loading || loader) return <HomeLoader />;
  if (widgetsQuery.isError)
    return (
      <div role="alert">
        Unable to load widgets.{" "}
        <button
          type="button"
          onClick={() => {
            void widgetsQuery.refetch();
          }}
        >
          Try again
        </button>
      </div>
    );

  return (
    <div className="relative flex h-full w-full flex-col gap-7">
      <ManageWidgetsModal
        workspaceSlug={workspaceSlug.toString()}
        isModalOpen={showWidgetSettings}
        handleOnClose={() => setShowWidgetSettings(false)}
      />
      {!isWikiApp && <NoProjectsEmptyState />}

      {isAnyWidgetEnabled ? (
        <div className="flex flex-col">
          {widgets.map(({ key, is_enabled: isEnabled }) => {
            const WidgetComponent = HOME_WIDGETS_LIST[key]?.component;
            if (!WidgetComponent || !isEnabled) return null;
            return (
              <div key={key} className="py-4">
                <WidgetComponent workspaceSlug={workspaceSlug.toString()} />
              </div>
            );
          })}
        </div>
      ) : (
        <div className="grid h-full w-full place-items-center">
          <SimpleEmptyState
            title={t("home.empty.widgets.title")}
            description={t("home.empty.widgets.description")}
            assetPath={noWidgetsResolvedPath}
          />
        </div>
      )}
    </div>
  );
};
