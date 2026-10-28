/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useRef, useState } from "react";
import { skipToken } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
// plane types
import { PageIcon, ProjectIcon, WorkItemsIcon } from "@plane/propel/icons";
import type { TActivityEntityData, THomeWidgetProps, TRecentActivityFilterKeys } from "@plane/types";
// plane ui
// components
import { ContentOverflowWrapper } from "@/components/core/content-overflow-HOC";
// plane web services
import { WorkspaceService } from "@/services/workspace.service";
import { RecentsEmptyState } from "../empty-states";
import { EWidgetKeys, WidgetLoader } from "../loaders";
import { FiltersDropdown } from "./filters";
import { RecentIssue } from "./issue";
import { RecentPage } from "./page";
import { RecentProject } from "./project";

const WIDGET_KEY = EWidgetKeys.RECENT_ACTIVITY;
const workspaceService = new WorkspaceService();
const filters: { name: TRecentActivityFilterKeys; icon?: React.ReactNode; i18n_key: string }[] = [
  { name: "all item", i18n_key: "home.recents.filters.all" },
  { name: "issue", icon: <WorkItemsIcon className="h-4 w-4" />, i18n_key: "home.recents.filters.issues" },
  { name: "page", icon: <PageIcon height={16} width={16} />, i18n_key: "home.recents.filters.pages" },
  { name: "project", icon: <ProjectIcon height={16} width={16} />, i18n_key: "home.recents.filters.projects" },
];

type TRecentWidgetProps = THomeWidgetProps & {
  presetFilter?: TRecentActivityFilterKeys;
  showFilterSelect?: boolean;
};

export const RecentActivityWidget = function RecentActivityWidget(props: TRecentWidgetProps) {
  const { presetFilter, showFilterSelect = true, workspaceSlug } = props;
  // states
  const [filter, setFilter] = useState<TRecentActivityFilterKeys>(presetFilter ?? filters[0].name);
  const { t } = useTranslation();
  // ref
  const ref = useRef<HTMLDivElement>(null);

  const { data: recents, isLoading } = useRecordCollection({
    queryKey: ["WORKSPACE_RECENT_ACTIVITY", workspaceSlug, filter],
    queryFn: workspaceSlug
      ? () =>
          workspaceService.fetchWorkspaceRecents(
            workspaceSlug.toString(),
            filter === filters[0].name ? undefined : filter
          )
      : skipToken,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });

  const resolveRecent = (activity: TActivityEntityData) => {
    switch (activity.entity_name) {
      case "page":
      case "workspace_page":
        return <RecentPage activity={activity} parentRef={ref} workspaceSlug={workspaceSlug} />;
      case "project":
        return <RecentProject activity={activity} parentRef={ref} workspaceSlug={workspaceSlug} />;
      case "issue":
        return <RecentIssue activity={activity} parentRef={ref} workspaceSlug={workspaceSlug} />;
      default:
        return <></>;
    }
  };

  if (!isLoading && recents?.length === 0)
    return (
      <div ref={ref} className="max-h-[500px] overflow-y-scroll">
        <div className="mb-4 flex items-center justify-between">
          <div className="text-14 font-semibold text-tertiary">{t("home.recents.title")}</div>
          {showFilterSelect && <FiltersDropdown filters={filters} activeFilter={filter} setActiveFilter={setFilter} />}
        </div>
        <div className="flex flex-col items-center justify-center">
          <RecentsEmptyState type={filter} />
        </div>
      </div>
    );

  return (
    <ContentOverflowWrapper
      maxHeight={415}
      containerClassName="box-border min-h-[250px]"
      fallback={<></>}
      buttonClassName="bg-surface-2/20"
    >
      <div className="mb-2 flex items-center justify-between">
        <div className="text-14 font-semibold text-tertiary">{t("home.recents.title")}</div>
        {showFilterSelect && <FiltersDropdown filters={filters} activeFilter={filter} setActiveFilter={setFilter} />}
      </div>
      <div className="flex min-h-[250px] flex-col">
        {isLoading && <WidgetLoader widgetKey={WIDGET_KEY} />}
        {!isLoading &&
          recents
            ?.filter((recent) => recent.entity_data)
            .map((activity) => <div key={activity.id}>{resolveRecent(activity)}</div>)}
      </div>
    </ContentOverflowWrapper>
  );
};
