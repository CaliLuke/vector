/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useTypeFilterConfig } from "./use-type-filter-config";
import { usePropertyFilterConfigs } from "./use-property-filter-configs";
import { AtSign, Briefcase } from "lucide-react";
// plane imports
import { Logo } from "@plane/propel/emoji-icon-picker";
import {
  CalendarLayoutIcon,
  CycleGroupIcon,
  CycleIcon,
  ModuleIcon,
  StatePropertyIcon,
  PriorityIcon,
  StateGroupIcon,
  MembersPropertyIcon,
  LabelPropertyIcon,
  StartDatePropertyIcon,
  DueDatePropertyIcon,
  UserCirclePropertyIcon,
  PriorityPropertyIcon,
} from "@plane/propel/icons";
import type { IState, IUserLite, TFilterConfig, IIssueLabel, TWorkItemFilterProperty } from "@plane/types";
import { Avatar } from "@plane/ui";
import {
  getAssigneeFilterConfig,
  getCreatedAtFilterConfig,
  getCreatedByFilterConfig,
  getCycleFilterConfig,
  getLabelFilterConfig,
  getMentionFilterConfig,
  getModuleFilterConfig,
  getPriorityFilterConfig,
  getProjectFilterConfig,
  getStartDateFilterConfig,
  getStateFilterConfig,
  getStateGroupFilterConfig,
  getSubscriberFilterConfig,
  getTargetDateFilterConfig,
  getUpdatedAtFilterConfig,
} from "@plane/utils/work-item-filters";
import { getFileURL } from "@plane/utils/file";
// store hooks
import { useWorkspaceCycles } from "@/hooks/use-cycles";
import { useWorkspaceLabels } from "@/hooks/use-labels";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useWorkspaceModules } from "@/hooks/use-modules";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkspaceStates } from "@/hooks/use-project-states";
// plane web imports
import { useFiltersOperatorConfigs } from "@/hooks/rich-filters/use-filters-operator-configs";

export type TWorkItemFiltersEntityProps = {
  workspaceSlug: string;
  cycleIds?: string[];
  labelIds?: string[];
  memberIds?: string[];
  moduleIds?: string[];
  projectId?: string;
  projectIds?: string[];
  stateIds?: string[];
};

export type TUseWorkItemFiltersConfigProps = {
  allowedFilters: TWorkItemFilterProperty[];
  propertyFiltersSupported?: boolean;
} & TWorkItemFiltersEntityProps;

export type TWorkItemFiltersConfig = {
  areAllConfigsInitialized: boolean;
  configs: TFilterConfig<TWorkItemFilterProperty>[];
  configMap: {
    [key in TWorkItemFilterProperty]?: TFilterConfig<TWorkItemFilterProperty>;
  };
  isFilterEnabled: (key: TWorkItemFilterProperty) => boolean;
  members: IUserLite[];
  catalogError: boolean;
  retryCatalogs: () => Promise<void>;
};

function filterCatalogError(
  filtersToShow: ReadonlySet<TWorkItemFilterProperty>,
  projectCatalogue: ReturnType<typeof useWorkspaceProjectDetails>,
  cycleRead: ReturnType<typeof useWorkspaceCycles>,
  moduleRead: ReturnType<typeof useWorkspaceModules>,
  typeFilter: ReturnType<typeof useTypeFilterConfig>,
  propertyFilters: ReturnType<typeof usePropertyFilterConfigs>
) {
  return (
    projectCatalogue.isError ||
    projectCatalogue.accessDenied ||
    typeFilter.isError ||
    propertyFilters.isError ||
    (filtersToShow.has("cycle_id") && (cycleRead.isError || cycleRead.isAccessDenied)) ||
    (filtersToShow.has("module_id") && (moduleRead.isError || moduleRead.isAccessDenied))
  );
}

function areFilterCatalogsInitialized(
  projectCatalogueReady: boolean,
  filtersToShow: ReadonlySet<TWorkItemFilterProperty>,
  cyclePending: boolean,
  modulePending: boolean
) {
  return (
    projectCatalogueReady &&
    (!filtersToShow.has("cycle_id") || !cyclePending) &&
    (!filtersToShow.has("module_id") || !modulePending)
  );
}

export const useWorkItemFiltersConfig = (props: TUseWorkItemFiltersConfigProps): TWorkItemFiltersConfig => {
  const { allowedFilters, cycleIds, labelIds, memberIds, moduleIds, projectId, projectIds, stateIds, workspaceSlug } =
    props;
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails(workspaceSlug);
  const projectRows = projectCatalogue.projects;
  const cycleRead = useWorkspaceCycles(workspaceSlug);
  const cycleCatalogue = cycleRead.cycles;
  const { getLabelById } = useWorkspaceLabels(workspaceSlug);
  const moduleRead = useWorkspaceModules(workspaceSlug);
  const moduleCatalogue = moduleRead.modules;
  const { getStateById } = useWorkspaceStates(workspaceSlug);
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  // derived values
  const operatorConfigs = useFiltersOperatorConfigs({ workspaceSlug });
  const filtersToShow = useMemo(() => new Set(allowedFilters), [allowedFilters]);
  const typeFilter = useTypeFilterConfig(workspaceSlug, projectId, filtersToShow.has("type_id"));
  const propertyFilters = usePropertyFilterConfigs(workspaceSlug, projectId, props.propertyFiltersSupported ?? false);
  const propertyKeys = useMemo(
    () => new Set(propertyFilters.configs.map((config) => config.id)),
    [propertyFilters.configs]
  );
  const project = useMemo(
    () => projectRows?.find((projectRow) => projectRow.id === projectId),
    [projectId, projectRows]
  );
  const members: IUserLite[] | undefined = useMemo(
    () =>
      memberIds
        ? memberIds.flatMap((memberId) => {
            const member = membersById?.get(memberId)?.member;
            return member ? [member] : [];
          })
        : undefined,
    [memberIds, membersById]
  );
  const workItemStates: IState[] | undefined = useMemo(
    () =>
      stateIds
        ? stateIds.flatMap((stateId) => {
            const state = getStateById(stateId);
            return state ? [state] : [];
          })
        : undefined,
    [stateIds, getStateById]
  );
  const workItemLabels: IIssueLabel[] | undefined = useMemo(
    () =>
      labelIds
        ? labelIds.flatMap((labelId) => {
            const label = getLabelById(labelId);
            return label ? [label] : [];
          })
        : undefined,
    [labelIds, getLabelById]
  );
  const cycles = useMemo(
    () => (cycleIds ? cycleIds.flatMap((id) => cycleCatalogue?.filter((row) => row.id === id) ?? []) : []),
    [cycleIds, cycleCatalogue]
  );
  const modules = useMemo(
    () => (moduleIds ? moduleIds.flatMap((id) => moduleCatalogue?.filter((row) => row.id === id) ?? []) : []),
    [moduleIds, moduleCatalogue]
  );
  const projects = useMemo(
    () =>
      projectIds ? projectIds.flatMap((id) => projectRows?.filter((projectRow) => projectRow.id === id) ?? []) : [],
    [projectIds, projectRows]
  );
  const areAllConfigsInitialized = !projectCatalogue.isPending;

  /**
   * Checks if a filter is enabled based on the filters to show.
   * @param key - The filter key.
   * @param level - The level of the filter.
   * @returns True if the filter is enabled, false otherwise.
   */
  const isFilterEnabled = useCallback(
    (key: TWorkItemFilterProperty) => filtersToShow.has(key) || propertyKeys.has(key),
    [filtersToShow, propertyKeys]
  );

  // state group filter config
  const stateGroupFilterConfig = useMemo(
    () =>
      getStateGroupFilterConfig<TWorkItemFilterProperty>("state_group")({
        isEnabled: isFilterEnabled("state_group"),
        filterIcon: StatePropertyIcon,
        getOptionIcon: (stateGroupKey) => <StateGroupIcon stateGroup={stateGroupKey} />,
        ...operatorConfigs,
      }),
    [isFilterEnabled, operatorConfigs]
  );

  // state filter config
  const stateFilterConfig = useMemo(
    () =>
      getStateFilterConfig<TWorkItemFilterProperty>("state_id")({
        isEnabled: isFilterEnabled("state_id") && workItemStates !== undefined,
        filterIcon: StatePropertyIcon,
        getOptionIcon: (state) => <StateGroupIcon stateGroup={state.group} color={state.color} />,
        states: workItemStates ?? [],
        ...operatorConfigs,
      }),
    [isFilterEnabled, workItemStates, operatorConfigs]
  );

  // label filter config
  const labelFilterConfig = useMemo(
    () =>
      getLabelFilterConfig<TWorkItemFilterProperty>("label_id")({
        isEnabled: isFilterEnabled("label_id") && workItemLabels !== undefined,
        filterIcon: LabelPropertyIcon,
        labels: workItemLabels ?? [],
        getOptionIcon: (color) => (
          <span className="flex size-2.5 flex-shrink-0 rounded-full" style={{ backgroundColor: color }} />
        ),
        ...operatorConfigs,
      }),
    [isFilterEnabled, workItemLabels, operatorConfigs]
  );

  // cycle filter config
  const cycleFilterConfig = useMemo(
    () =>
      getCycleFilterConfig<TWorkItemFilterProperty>("cycle_id")({
        isEnabled: isFilterEnabled("cycle_id") && project?.cycle_view === true && cycles !== undefined,
        filterIcon: CycleIcon,
        getOptionIcon: (cycleGroup) => <CycleGroupIcon cycleGroup={cycleGroup} className="h-3.5 w-3.5 flex-shrink-0" />,
        cycles: cycles ?? [],
        ...operatorConfigs,
      }),
    [isFilterEnabled, project?.cycle_view, cycles, operatorConfigs]
  );

  // module filter config
  const moduleFilterConfig = useMemo(
    () =>
      getModuleFilterConfig<TWorkItemFilterProperty>("module_id")({
        isEnabled: isFilterEnabled("module_id") && project?.module_view === true && modules !== undefined,
        filterIcon: ModuleIcon,
        getOptionIcon: () => <ModuleIcon className="h-3 w-3 flex-shrink-0" />,
        modules: modules ?? [],
        ...operatorConfigs,
      }),
    [isFilterEnabled, project?.module_view, modules, operatorConfigs]
  );

  // assignee filter config
  const assigneeFilterConfig = useMemo(
    () =>
      getAssigneeFilterConfig<TWorkItemFilterProperty>("assignee_id")({
        isEnabled: isFilterEnabled("assignee_id") && members !== undefined,
        filterIcon: MembersPropertyIcon,
        members: members ?? [],
        getOptionIcon: (memberDetails) => (
          <Avatar
            name={memberDetails.display_name}
            src={getFileURL(memberDetails.avatar_url)}
            showTooltip={false}
            size="sm"
          />
        ),
        ...operatorConfigs,
      }),
    [isFilterEnabled, members, operatorConfigs]
  );

  // mention filter config
  const mentionFilterConfig = useMemo(
    () =>
      getMentionFilterConfig<TWorkItemFilterProperty>("mention_id")({
        isEnabled: isFilterEnabled("mention_id") && members !== undefined,
        filterIcon: AtSign,
        members: members ?? [],
        getOptionIcon: (memberDetails) => (
          <Avatar
            name={memberDetails.display_name}
            src={getFileURL(memberDetails.avatar_url)}
            showTooltip={false}
            size="sm"
          />
        ),
        ...operatorConfigs,
      }),
    [isFilterEnabled, members, operatorConfigs]
  );

  // created by filter config
  const createdByFilterConfig = useMemo(
    () =>
      getCreatedByFilterConfig<TWorkItemFilterProperty>("created_by_id")({
        isEnabled: isFilterEnabled("created_by_id") && members !== undefined,
        filterIcon: UserCirclePropertyIcon,
        members: members ?? [],
        getOptionIcon: (memberDetails) => (
          <Avatar
            name={memberDetails.display_name}
            src={getFileURL(memberDetails.avatar_url)}
            showTooltip={false}
            size="sm"
          />
        ),
        ...operatorConfigs,
      }),
    [isFilterEnabled, members, operatorConfigs]
  );

  // subscriber filter config
  const subscriberFilterConfig = useMemo(
    () =>
      getSubscriberFilterConfig<TWorkItemFilterProperty>("subscriber_id")({
        isEnabled: isFilterEnabled("subscriber_id") && members !== undefined,
        filterIcon: MembersPropertyIcon,
        members: members ?? [],
        getOptionIcon: (memberDetails) => (
          <Avatar
            name={memberDetails.display_name}
            src={getFileURL(memberDetails.avatar_url)}
            showTooltip={false}
            size="sm"
          />
        ),
        ...operatorConfigs,
      }),
    [isFilterEnabled, members, operatorConfigs]
  );

  // priority filter config
  const priorityFilterConfig = useMemo(
    () =>
      getPriorityFilterConfig<TWorkItemFilterProperty>("priority")({
        isEnabled: isFilterEnabled("priority"),
        filterIcon: PriorityPropertyIcon,
        getOptionIcon: (priority) => <PriorityIcon priority={priority} />,
        ...operatorConfigs,
      }),
    [isFilterEnabled, operatorConfigs]
  );

  // start date filter config
  const startDateFilterConfig = useMemo(
    () =>
      getStartDateFilterConfig<TWorkItemFilterProperty>("start_date")({
        isEnabled: true,
        filterIcon: StartDatePropertyIcon,
        ...operatorConfigs,
      }),
    [operatorConfigs]
  );

  // target date filter config
  const targetDateFilterConfig = useMemo(
    () =>
      getTargetDateFilterConfig<TWorkItemFilterProperty>("target_date")({
        isEnabled: true,
        filterIcon: DueDatePropertyIcon,
        ...operatorConfigs,
      }),
    [operatorConfigs]
  );

  // created at filter config
  const createdAtFilterConfig = useMemo(
    () =>
      getCreatedAtFilterConfig<TWorkItemFilterProperty>("created_at")({
        isEnabled: true,
        filterIcon: CalendarLayoutIcon,
        ...operatorConfigs,
      }),
    [operatorConfigs]
  );

  // updated at filter config
  const updatedAtFilterConfig = useMemo(
    () =>
      getUpdatedAtFilterConfig<TWorkItemFilterProperty>("updated_at")({
        isEnabled: true,
        filterIcon: CalendarLayoutIcon,
        ...operatorConfigs,
      }),
    [operatorConfigs]
  );

  // project filter config
  const projectFilterConfig = useMemo(
    () =>
      getProjectFilterConfig<TWorkItemFilterProperty>("project_id")({
        isEnabled: isFilterEnabled("project_id") && projects !== undefined,
        filterIcon: Briefcase,
        projects: projects,
        getOptionIcon: (option) => <Logo logo={option.logo_props} size={12} />,
        ...operatorConfigs,
      }),
    [isFilterEnabled, projects, operatorConfigs]
  );

  return {
    areAllConfigsInitialized: areFilterCatalogsInitialized(
      areAllConfigsInitialized,
      filtersToShow,
      cycleRead.isPending,
      moduleRead.isPending
    ),
    catalogError: filterCatalogError(
      filtersToShow,
      projectCatalogue,
      cycleRead,
      moduleRead,
      typeFilter,
      propertyFilters
    ),
    retryCatalogs: async () => {
      await Promise.all([
        projectCatalogue.isError || projectCatalogue.accessDenied ? projectCatalogue.retryAccess() : undefined,
        filtersToShow.has("cycle_id") ? cycleRead.retry() : undefined,
        filtersToShow.has("module_id") ? moduleRead.retry() : undefined,
        typeFilter.isError ? typeFilter.retry() : undefined,
        propertyFilters.isError ? propertyFilters.retry() : undefined,
      ]);
    },
    configs: [
      typeFilter.config,
      ...propertyFilters.configs,
      stateFilterConfig,
      stateGroupFilterConfig,
      assigneeFilterConfig,
      priorityFilterConfig,
      projectFilterConfig,
      mentionFilterConfig,
      labelFilterConfig,
      cycleFilterConfig,
      moduleFilterConfig,
      startDateFilterConfig,
      targetDateFilterConfig,
      createdAtFilterConfig,
      updatedAtFilterConfig,
      createdByFilterConfig,
      subscriberFilterConfig,
    ],
    configMap: {
      ...Object.fromEntries(propertyFilters.configs.map((config) => [config.id, config])),
      type_id: typeFilter.config,
      project_id: projectFilterConfig,
      state_group: stateGroupFilterConfig,
      state_id: stateFilterConfig,
      label_id: labelFilterConfig,
      cycle_id: cycleFilterConfig,
      module_id: moduleFilterConfig,
      assignee_id: assigneeFilterConfig,
      mention_id: mentionFilterConfig,
      created_by_id: createdByFilterConfig,
      subscriber_id: subscriberFilterConfig,
      priority: priorityFilterConfig,
      start_date: startDateFilterConfig,
      target_date: targetDateFilterConfig,
      created_at: createdAtFilterConfig,
      updated_at: updatedAtFilterConfig,
    },
    isFilterEnabled,
    members: members ?? [],
  };
};
