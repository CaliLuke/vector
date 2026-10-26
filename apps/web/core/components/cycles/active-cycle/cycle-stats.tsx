/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIssueLayoutLabels } from "@/hooks/use-issue-layout-labels";
import { useIssueEstimateOrder } from "@/hooks/use-issue-estimate-projection";
import { useIssueStateOrder } from "@/hooks/use-project-states";
import { useIssueParentCatalogues } from "@/hooks/use-issue-parent-catalogues";
import { projectIssueIdsByDisplayOrder } from "@/components/issues/issue-layouts/issue-display-ordering";
import { useProjectDetail } from "@/hooks/use-projects";
import { useCyclePriorityIssues } from "@/hooks/use-cycle-priority-issues";
import { useOptionalParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { Fragment, useCallback, useRef, useState } from "react";
import isEmpty from "lodash-es/isEmpty.js";

import { useTheme } from "next-themes";
import { CalendarCheck } from "lucide-react";
// headless ui
import { Tab } from "@headlessui/react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { PriorityIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TWorkItemFilterCondition } from "@plane/shared-state";
import type { ICycle } from "@plane/types";
// ui
import { Loader, Avatar } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { renderFormattedDate, renderFormattedDateWithoutYear } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
// assets
import darkAssigneeAsset from "@/app/assets/empty-state/active-cycle/assignee-dark.webp?url";
import lightAssigneeAsset from "@/app/assets/empty-state/active-cycle/assignee-light.webp?url";
import darkLabelAsset from "@/app/assets/empty-state/active-cycle/label-dark.webp?url";
import lightLabelAsset from "@/app/assets/empty-state/active-cycle/label-light.webp?url";
import darkPriorityAsset from "@/app/assets/empty-state/active-cycle/priority-dark.webp?url";
import lightPriorityAsset from "@/app/assets/empty-state/active-cycle/priority-light.webp?url";
import userImage from "@/app/assets/user.png?url";
// components
import { SingleProgressStats } from "@/components/core/sidebar/single-progress-stats";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
import { useParentChartPreferences } from "@/providers/cycle-module-preferences";
// store

export type ActiveCycleStatsProps = {
  workspaceSlug: string;
  projectId: string;
  cycle: ICycle | null;
  cycleId?: string | null;
  handleFiltersUpdate: (conditions: TWorkItemFilterCondition[]) => void;
};

function getStatisticsTabIndex(value: string | null) {
  switch (value) {
    case "Assignees":
      return 1;
    case "Labels":
      return 2;
    default:
      return 0;
  }
}

type TPriorityIssue = NonNullable<ReturnType<typeof useCyclePriorityIssues>["issues"]>[number];
type TAssigneeStats = NonNullable<NonNullable<ICycle["distribution"]>["assignees"]>[number];
type TLabelStats = NonNullable<NonNullable<ICycle["distribution"]>["labels"]>[number];

type PriorityIssueRowProps = {
  issue: TPriorityIssue;
  projectId: string;
  projectIdentifier: string;
  onOpen: (issue: TPriorityIssue) => void;
};

function PriorityIssueRow(props: PriorityIssueRowProps) {
  const { issue, projectId, projectIdentifier, onOpen } = props;
  return (
    <div className="group flex items-center justify-between gap-2 rounded-md p-1 hover:bg-surface-2">
      <button
        type="button"
        className="flex w-full min-w-24 flex-grow cursor-pointer items-center gap-1.5 truncate text-left"
        onClick={() => onOpen(issue)}
      >
        <IssueIdentifier
          projectIdentifier={projectIdentifier}
          issueSequenceId={issue.sequence_id}
          issueTypeId={issue.type_id}
          projectId={projectId}
          size="xs"
          variant="secondary"
        />
        <Tooltip position="top-start" tooltipHeading="Title" tooltipContent={issue.name}>
          <span className="truncate text-13 text-primary">{issue.name}</span>
        </Tooltip>
        <PriorityIcon priority={issue.priority} withContainer size={12} />
      </button>
      <div className="flex flex-shrink-0 items-center gap-1.5">
        <StateDropdown
          value={issue.state_id}
          onChange={() => {}}
          projectId={projectId}
          disabled
          buttonVariant="background-with-text"
          buttonContainerClassName="cursor-pointer max-w-24"
          showTooltip
        />
        {issue.target_date && (
          <Tooltip tooltipHeading="Target Date" tooltipContent={renderFormattedDate(issue.target_date)}>
            <div className="flex h-full items-center gap-1.5 truncate rounded-sm bg-layer-1 px-2 py-0.5 text-11 group-hover:bg-surface-1">
              <CalendarCheck className="h-3 w-3 flex-shrink-0" />
              <span className="truncate text-11">{renderFormattedDateWithoutYear(issue.target_date)}</span>
            </div>
          </Tooltip>
        )}
      </div>
    </div>
  );
}

type PriorityIssuesPanelProps = {
  issues: readonly TPriorityIssue[] | undefined;
  issueCount: number;
  hasNextPage: boolean;
  isError: boolean;
  isPending: boolean;
  retry: () => Promise<void>;
  projectId: string;
  projectIdentifier: string;
  loaderRef: (element: HTMLDivElement | null) => void;
  containerRef: React.RefObject<HTMLDivElement | null>;
  emptyImage: string;
  translate: ReturnType<typeof useTranslation>["t"];
  onOpenIssue: (issue: TPriorityIssue) => void;
};

function CyclePriorityIssuesPanel(props: PriorityIssuesPanelProps) {
  const {
    issues,
    issueCount,
    hasNextPage,
    isError,
    isPending,
    retry,
    projectId,
    projectIdentifier,
    loaderRef,
    containerRef,
    emptyImage,
    translate,
    onOpenIssue,
  } = props;
  const loaders = (
    <Loader className="space-y-3">
      <Loader.Item height="30px" />
      <Loader.Item height="30px" />
      <Loader.Item height="30px" />
    </Loader>
  );
  if (isError) return <CatalogueReadFailure label="priority work items" retry={retry} />;
  if (isPending || !issues) return loaders;
  if (issueCount === 0)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <SimpleEmptyState title={translate("active_cycle.empty_state.priority_issue.title")} assetPath={emptyImage} />
      </div>
    );

  return (
    <div ref={containerRef} className="vertical-scrollbar flex h-full w-full flex-col gap-1 overflow-y-auto">
      {issues.map((issue) => (
        <PriorityIssueRow
          key={issue.id}
          issue={issue}
          projectId={projectId}
          projectIdentifier={projectIdentifier}
          onOpen={onOpenIssue}
        />
      ))}
      {hasNextPage && (
        <div ref={loaderRef} className="relative flex h-11 animate-pulse items-center gap-3 bg-layer-1 p-3 text-13" />
      )}
    </div>
  );
}

function CycleAssigneeStatsPanel(props: {
  cycle: ICycle | null;
  assignees: readonly TAssigneeStats[] | undefined;
  emptyImage: string;
  translate: ReturnType<typeof useTranslation>["t"];
  onFilter: (condition: TWorkItemFilterCondition[]) => void;
}) {
  const { cycle, assignees, emptyImage, translate, onFilter } = props;
  if (!cycle || isEmpty(cycle.distribution))
    return (
      <Loader className="space-y-3">
        <Loader.Item height="30px" />
      </Loader>
    );
  if (!assignees?.length)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <SimpleEmptyState title={translate("active_cycle.empty_state.assignee.title")} assetPath={emptyImage} />
      </div>
    );
  return assignees.map((assignee) => {
    const assigneeId = assignee.assignee_id;
    return (
      <SingleProgressStats
        key={assigneeId ?? "unassigned"}
        title={
          <div className="flex items-center gap-2">
            {assigneeId ? (
              <Avatar name={assignee.display_name ?? undefined} src={getFileURL(assignee.avatar_url ?? "")} />
            ) : (
              <div className="h-5 w-5 rounded-full border-2 border-subtle bg-layer-1">
                <img src={userImage} height="100%" width="100%" className="rounded-full" alt="User" />
              </div>
            )}
            <span>{assignee.display_name ?? translate("no_assignee")}</span>
          </div>
        }
        completed={assignee.completed_issues}
        total={assignee.total_issues}
        onClick={
          assigneeId ? () => onFilter([{ property: "assignee_id", operator: "in", value: [assigneeId] }]) : undefined
        }
      />
    );
  });
}

function CycleLabelStatsPanel(props: {
  cycle: ICycle | null;
  labels: readonly TLabelStats[] | undefined;
  emptyImage: string;
  translate: ReturnType<typeof useTranslation>["t"];
  onFilter: (condition: TWorkItemFilterCondition[]) => void;
}) {
  const { cycle, labels, emptyImage, translate, onFilter } = props;
  if (!cycle || isEmpty(cycle.distribution))
    return (
      <Loader className="space-y-3">
        <Loader.Item height="30px" />
      </Loader>
    );
  if (!labels?.length)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <SimpleEmptyState title={translate("active_cycle.empty_state.label.title")} assetPath={emptyImage} />
      </div>
    );
  return labels.map((label) => {
    const labelId = label.label_id;
    return (
      <SingleProgressStats
        key={`${labelId ?? "no-label"}-${label.label_name ?? ""}`}
        title={
          <div className="flex items-center gap-2 truncate">
            <span
              className="block h-3 w-3 flex-shrink-0 rounded-full"
              style={{ backgroundColor: label.color ?? "#000000" }}
            />
            <span className="truncate text-11 text-ellipsis">{label.label_name ?? "No labels"}</span>
          </div>
        }
        completed={label.completed_issues}
        total={label.total_issues}
        onClick={labelId ? () => onFilter([{ property: "label_id", operator: "in", value: [labelId] }]) : undefined}
      />
    );
  });
}

type TStatisticsTab = "Priority-Issues" | "Assignees" | "Labels";

function getStatisticsTabValue(index: number): TStatisticsTab {
  if (index === 1) return "Assignees";
  if (index === 2) return "Labels";
  return "Priority-Issues";
}

function getStatisticsTabClassName({ selected }: { selected: boolean }) {
  return cn(
    "relative z-[1] rounded-[3px] py-1.5 text-11 font-semibold text-placeholder transition duration-500 focus:outline-none",
    {
      "bg-surface-1 text-tertiary": selected,
      "hover:text-tertiary": !selected,
    }
  );
}

type CycleStatsTabsProps = {
  tab: TStatisticsTab | null;
  onChange: (value: TStatisticsTab) => void;
  priorityPanel: React.ReactNode;
  assigneePanel: React.ReactNode;
  labelPanel: React.ReactNode;
  translate: ReturnType<typeof useTranslation>["t"];
};

function CycleStatsTabs(props: CycleStatsTabsProps) {
  const { tab, onChange, priorityPanel, assigneePanel, labelPanel, translate } = props;
  return (
    <Tab.Group
      as={Fragment}
      defaultIndex={getStatisticsTabIndex(tab)}
      onChange={(index) => onChange(getStatisticsTabValue(index))}
    >
      <Tab.List
        as="div"
        className="relative grid rounded-sm border-[0.5px] border-subtle bg-layer-1 p-[1px]"
        style={{ gridTemplateColumns: "repeat(3, 1fr)" }}
      >
        <Tab className={getStatisticsTabClassName}>{translate("project_cycles.active_cycle.priority_issue")}</Tab>
        <Tab className={getStatisticsTabClassName}>{translate("project_cycles.active_cycle.assignees")}</Tab>
        <Tab className={getStatisticsTabClassName}>{translate("project_cycles.active_cycle.labels")}</Tab>
      </Tab.List>
      <Tab.Panels as={Fragment}>
        <Tab.Panel
          as="div"
          className="vertical-scrollbar flex h-52 w-full flex-col gap-1 overflow-y-auto text-secondary"
        >
          {priorityPanel}
        </Tab.Panel>
        <Tab.Panel
          as="div"
          className="vertical-scrollbar flex h-52 w-full flex-col gap-1 overflow-y-auto text-secondary"
        >
          {assigneePanel}
        </Tab.Panel>
        <Tab.Panel
          as="div"
          className="vertical-scrollbar flex h-52 w-full flex-col gap-1 overflow-y-auto text-secondary"
        >
          {labelPanel}
        </Tab.Panel>
      </Tab.Panels>
    </Tab.Group>
  );
}

function getCycleOrderBy(
  preferences: ReturnType<typeof useOptionalParentIssuePreferences>,
  cycleId: string | null | undefined
) {
  if (preferences?.domain !== "cycle" || preferences.requestInput?.viewId !== cycleId) return undefined;
  return preferences.filters?.displayFilters?.order_by;
}

function getCycleOrderingStatus(props: {
  estimatesError: boolean;
  statesError: boolean;
  parentsError: boolean;
  labelsRequired: boolean;
  labelsError: boolean;
  labelsDenied: boolean;
  labelsScopeError: boolean;
  estimatesPending: boolean;
  statesPending: boolean;
  parentsPending: boolean;
  labelsPending: boolean;
  labelsScopeReady: boolean;
}) {
  const {
    estimatesError,
    statesError,
    parentsError,
    labelsRequired,
    labelsError,
    labelsDenied,
    labelsScopeError,
    estimatesPending,
    statesPending,
    parentsPending,
    labelsPending,
    labelsScopeReady,
  } = props;
  return {
    isError:
      estimatesError ||
      statesError ||
      parentsError ||
      (labelsRequired && (labelsError || labelsDenied || labelsScopeError)),
    isPending:
      estimatesPending || statesPending || parentsPending || (labelsRequired && (labelsPending || !labelsScopeReady)),
  };
}

function orderCyclePriorityIssues(
  issues: ReturnType<typeof useCyclePriorityIssues>["issues"],
  membersById: ReturnType<typeof useWorkspaceMemberRoster>["membersById"],
  orderBy: ReturnType<typeof getCycleOrderBy>,
  labelsById: ReturnType<typeof useIssueLayoutLabels>["labelsById"],
  estimates: ReturnType<typeof useIssueEstimateOrder>["projection"],
  states: ReturnType<typeof useIssueStateOrder>["projection"],
  parents: ReturnType<typeof useIssueParentCatalogues>["parents"]
) {
  const records = new Map(issues?.map((issue) => [issue.id, issue]));
  const orderedIds = projectIssueIdsByDisplayOrder(
    issues?.map((issue) => issue.id),
    (id) => records.get(id),
    membersById,
    orderBy,
    labelsById,
    estimates,
    states,
    parents
  );
  return orderedIds?.flatMap((id) => {
    const issue = records.get(id);
    return issue ? [issue] : [];
  });
}

async function retryCycleOrdering(
  retries: Array<() => Promise<unknown>>,
  labelRetry: () => Promise<unknown>,
  labelsRequired: boolean
) {
  await Promise.all([...retries, ...(labelsRequired ? [labelRetry] : [])].map((retry) => retry()));
}

function fetchNextPriorityPage(query: ReturnType<typeof useCyclePriorityIssues>["query"]) {
  if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
}

function useCyclePriorityPanelData(workspaceSlug: string, projectId: string, cycleId: string | null | undefined) {
  const preferences = useOptionalParentIssuePreferences();
  const orderBy = getCycleOrderBy(preferences, cycleId);
  const { project } = useProjectDetail(workspaceSlug, projectId);
  const priority = useCyclePriorityIssues(workspaceSlug, projectId, cycleId ?? undefined, orderBy);
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const estimates = useIssueEstimateOrder(workspaceSlug, projectId, false, orderBy);
  const states = useIssueStateOrder(workspaceSlug, projectId, false, orderBy);
  const parents = useIssueParentCatalogues(workspaceSlug, projectId, false, { order_by: orderBy });
  const labels = useIssueLayoutLabels(workspaceSlug, projectId, false);
  const labelsRequired = orderBy === "labels__name" || orderBy === "-labels__name";
  const ordering = getCycleOrderingStatus({
    estimatesError: estimates.isError,
    statesError: states.isError,
    parentsError: parents.isError,
    labelsRequired,
    labelsError: labels.isError,
    labelsDenied: labels.isAccessDenied,
    labelsScopeError: labels.scopeIsError,
    estimatesPending: estimates.isPending,
    statesPending: states.isPending,
    parentsPending: parents.isPending,
    labelsPending: labels.isPending,
    labelsScopeReady: labels.scopeIsReady,
  });
  const orderedIssues = orderCyclePriorityIssues(
    priority.issues,
    membersById,
    orderBy,
    labels.labelsById,
    estimates.projection,
    states.projection,
    parents.parents
  );
  const retryOrdering = () =>
    retryCycleOrdering([estimates.retry, states.retry, parents.retry], labels.retry, labelsRequired);

  return { project, priority, orderedIssues, ordering, retryOrdering };
}

type CyclePriorityPanelProps = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string | null | undefined;
  emptyImage: string;
  translate: ReturnType<typeof useTranslation>["t"];
  onFilter: (conditions: TWorkItemFilterCondition[]) => void;
};

function CyclePriorityPanel(props: CyclePriorityPanelProps) {
  const { workspaceSlug, projectId, cycleId, emptyImage, translate, onFilter } = props;
  const panelData = useCyclePriorityPanelData(workspaceSlug, projectId, cycleId);
  const { priority } = panelData;
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [loaderElement, setLoaderElement] = useState<HTMLDivElement | null>(null);
  const { setPeekIssue } = useIssueDetailUiState();
  const openIssue = (issue: TPriorityIssue) => {
    setPeekIssue({ workspaceSlug, projectId, issueId: issue.id, isArchived: !!issue.archived_at });
    onFilter([{ property: "priority", operator: "in", value: ["urgent", "high"] }]);
  };
  const loadMore = useCallback(() => fetchNextPriorityPage(priority.query), [priority.query]);
  useIntersectionObserver(containerRef, loaderElement, loadMore, "0% 0% 100% 0%");
  const retry = async () => {
    await Promise.all([
      ...(priority.isError || priority.isAccessDenied ? [priority.retry()] : []),
      panelData.retryOrdering(),
    ]);
  };

  return (
    <CyclePriorityIssuesPanel
      issues={panelData.orderedIssues}
      issueCount={priority.query.data?.pages.at(-1)?.total_count ?? 0}
      hasNextPage={priority.query.hasNextPage}
      isError={priority.isError || priority.isAccessDenied || panelData.ordering.isError}
      isPending={panelData.ordering.isPending || !priority.issues}
      retry={retry}
      projectId={projectId}
      projectIdentifier={panelData.project?.identifier ?? ""}
      loaderRef={setLoaderElement}
      containerRef={containerRef}
      emptyImage={emptyImage}
      translate={translate}
      onOpenIssue={openIssue}
    />
  );
}

export const ActiveCycleStats = function ActiveCycleStats(props: ActiveCycleStatsProps) {
  const { workspaceSlug, projectId, cycle, cycleId, handleFiltersUpdate } = props;
  // local storage
  const chart = useParentChartPreferences("cycle", workspaceSlug, projectId, cycleId ?? undefined);
  const tab = chart.statisticsTab;
  const setTab = (statisticsTab: typeof chart.statisticsTab) => chart.update({ statisticsTab });
  // theme hook
  const { resolvedTheme } = useTheme();
  // plane hooks
  const { t } = useTranslation();
  // derived values
  const priorityResolvedPath = resolvedTheme === "light" ? lightPriorityAsset : darkPriorityAsset;
  const assigneesResolvedPath = resolvedTheme === "light" ? lightAssigneeAsset : darkAssigneeAsset;
  const labelsResolvedPath = resolvedTheme === "light" ? lightLabelAsset : darkLabelAsset;

  const distribution = cycle?.distribution;
  const assigneeDistribution = distribution?.assignees;
  const labelDistribution = distribution?.labels;

  if (!cycleId)
    return (
      <Loader className="col-span-1 flex min-h-[17rem] flex-col gap-4 overflow-hidden bg-surface-1 lg:col-span-2 xl:col-span-1">
        <Loader.Item width="100%" height="17rem" />
      </Loader>
    );

  const priorityPanel = (
    <CyclePriorityPanel
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      cycleId={cycleId}
      emptyImage={priorityResolvedPath}
      translate={t}
      onFilter={handleFiltersUpdate}
    />
  );
  const assigneePanel = (
    <CycleAssigneeStatsPanel
      cycle={cycle}
      assignees={assigneeDistribution}
      emptyImage={assigneesResolvedPath}
      translate={t}
      onFilter={handleFiltersUpdate}
    />
  );
  const labelPanel = (
    <CycleLabelStatsPanel
      cycle={cycle}
      labels={labelDistribution}
      emptyImage={labelsResolvedPath}
      translate={t}
      onFilter={handleFiltersUpdate}
    />
  );

  return (
    <div className="col-span-1 flex min-h-[17rem] flex-col gap-4 overflow-hidden rounded-lg border border-subtle bg-surface-1 p-4 lg:col-span-2 xl:col-span-1">
      <CycleStatsTabs
        tab={tab}
        onChange={setTab}
        priorityPanel={priorityPanel}
        assigneePanel={assigneePanel}
        labelPanel={labelPanel}
        translate={t}
      />
    </div>
  );
};
