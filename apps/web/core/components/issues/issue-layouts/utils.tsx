// oxlint-disable no-shadow
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { CSSProperties } from "react";
import { isParentTypeGrouping, isCustomGrouping } from "@/helpers/issue-list/grouping-axis";
import { extractInstruction } from "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item";
import clone from "lodash-es/clone.js";
import isNil from "lodash-es/isNil.js";
import scrollIntoView from "smooth-scroll-into-view-if-needed";
import type { FC } from "react";
import { CalendarDays, LayersIcon, Paperclip } from "lucide-react";
// plane types
import { EIconSize, ISSUE_PRIORITIES, STATE_GROUPS } from "@plane/constants";
import { Logo } from "@plane/propel/emoji-icon-picker";
import type { ISvgIcons } from "@plane/propel/icons";
import {
  CycleGroupIcon,
  CycleIcon,
  ModuleIcon,
  PriorityIcon,
  StateGroupIcon,
  LinkIcon,
  StatePropertyIcon,
  MembersPropertyIcon,
  DueDatePropertyIcon,
  EstimatePropertyIcon,
  LabelPropertyIcon,
  PriorityPropertyIcon,
  StartDatePropertyIcon,
} from "@plane/propel/icons";
import type {
  GroupByColumnTypes,
  IGroupByColumn,
  TCycleGroups,
  IIssueDisplayProperties,
  TIssueStandardDisplayProperty,
  IPragmaticDropPayload,
  TIssue,
  TIssueGroupByOptions,
  IIssueFilterOptions,
  IIssueFilters,
  IIssueDisplayFilterOptions,
  TGetColumns,
  IWorkspaceMember,
  IIssueLabel,
  IState,
  IProject,
  ICycle,
  IModule,
  TSpreadsheetColumn,
} from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
// plane ui
import { Avatar } from "@plane/ui";
import { renderFormattedDate } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
// store
import { ISSUE_FILTER_DEFAULT_DATA } from "@/helpers/issue-list/properties";
import { DEFAULT_SUB_WORK_ITEM_DISPLAY_PROPERTIES } from "@/helpers/sub-work-item-preferences";
// constants
import { ISSUE_GROUP_BY_OPTIONS } from "@plane/constants";
// components
import {
  SpreadsheetAssigneeColumn,
  SpreadsheetAttachmentColumn,
  SpreadsheetCreatedOnColumn,
  SpreadsheetDueDateColumn,
  SpreadsheetEstimateColumn,
  SpreadsheetLabelColumn,
  SpreadsheetModuleColumn,
  SpreadsheetCycleColumn,
  SpreadsheetLinkColumn,
  SpreadsheetPriorityColumn,
  SpreadsheetStartDateColumn,
  SpreadsheetStateColumn,
  SpreadsheetSubIssueColumn,
  SpreadsheetUpdatedOnColumn,
} from "@/components/issues/issue-layouts/spreadsheet/columns";

export const HIGHLIGHT_CLASS = "highlight";
export const HIGHLIGHT_WITH_LINE = "highlight-with-line";

export type GroupDropLocation = {
  columnId: string;
  groupId: string;
  subGroupId?: string;
  id: string | undefined;
  canAddIssueBelow?: boolean;
};

export type IssueUpdates = {
  [groupKey: string]: {
    ADD: string[];
    REMOVE: string[];
  };
};

export const isWorkspaceLevel = (type: EIssuesStoreType) =>
  // oxlint-disable-next-line no-unneeded-ternary
  [
    EIssuesStoreType.PROFILE,
    EIssuesStoreType.GLOBAL,
    EIssuesStoreType.TEAM,
    EIssuesStoreType.TEAM_VIEW,
    EIssuesStoreType.TEAM_PROJECT_WORK_ITEMS,
  ].includes(type)
    ? true
    : false;

export function isSubscribedProfileRequest(requestInput: unknown) {
  if (typeof requestInput !== "object" || requestInput === null || !("kind" in requestInput)) return false;
  return requestInput.kind === "profile" && "view" in requestInput && requestInput.view === "subscribed";
}

type TGetGroupByColumns = {
  groupBy: GroupByColumnTypes | null;
  includeNone: boolean;
  isEpic?: boolean;
  projectId?: string;
};

// NOTE: Type of groupBy is different compared to what's being passed from the components.
// We are using `as` to typecast it to the expected type.
// It can break the includeNone logic if not handled properly.
type TGetGroupByColumnsWithIdentity = TGetGroupByColumns & {
  assigneeIds: readonly string[];
  creatorIds: readonly string[];
  membersById: ReadonlyMap<string, IWorkspaceMember> | undefined;
  joinedProjectIds: readonly string[];
  labels: readonly IIssueLabel[] | undefined;
  typeColumns?: IGroupByColumn[];
  states?: readonly IState[];
  projects?: readonly Pick<IProject, "id" | "name" | "logo_props">[];
  cycles?: readonly ICycle[];
  modules?: readonly IModule[];
};

export const getGroupByColumns = ({
  groupBy,
  includeNone,
  isEpic = false,
  projectId,
  assigneeIds,
  creatorIds,
  membersById,
  joinedProjectIds,
  labels,
  typeColumns,
  states,
  projects,
  cycles,
  modules,
}: TGetGroupByColumnsWithIdentity): IGroupByColumn[] | undefined => {
  // If no groupBy is specified and includeNone is true, return "All Issues" group
  if (!groupBy && includeNone) {
    return [
      {
        id: "All Issues",
        name: `All ${isEpic ? "Epics" : "work items"}`,
        payload: {},
        icon: undefined,
      },
    ];
  }

  // Return undefined if no valid groupBy
  if (!groupBy || isCustomGrouping(groupBy) || isParentTypeGrouping(groupBy)) return undefined;

  // Map of group by options to their corresponding column getter functions
  const groupByColumnMap: Record<
    Exclude<GroupByColumnTypes, `customproperty_${string}` | `parent_type_${string}`>,
    () => IGroupByColumn[] | undefined
  > = {
    type_id: () => typeColumns,
    project: () => getProjectColumns(projects, joinedProjectIds),
    cycle: () => getCycleColumns(cycles),
    module: () => getModuleColumns(modules),
    state: () => getStateColumns({ projectId, states }),
    "state_detail.group": () => getStateGroupColumns(),
    priority: () => getPriorityColumns(),
    labels: () => getLabelsColumns(labels),
    assignees: () => getMemberColumns(assigneeIds, membersById, "assignees"),
    created_by: () => getMemberColumns(creatorIds, membersById, "created_by"),
    team_project: () => getTeamProjectColumns(),
  };

  // Get and return the columns for the specified group by option
  return groupByColumnMap[groupBy]?.();
};

const getProjectColumns = (
  projects: readonly Pick<IProject, "id" | "name" | "logo_props">[] | undefined,
  projectIds: readonly string[]
): IGroupByColumn[] | undefined =>
  projects
    ?.filter((project) => projectIds.includes(project.id))
    .map((project) => ({
      id: project.id,
      name: project.name,
      icon: (
        <div className="grid h-6 w-6 flex-shrink-0 place-items-center">
          <Logo logo={project.logo_props} />
        </div>
      ),
      payload: { project_id: project.id },
    }));

const getCycleColumns = (cycleDetails: readonly ICycle[] | undefined): IGroupByColumn[] | undefined => {
  if (!cycleDetails) return undefined;
  const cycles: IGroupByColumn[] = [];
  cycleDetails?.map((cycle) => {
    const cycleStatus = cycle.status ? (cycle.status.toLocaleLowerCase() as TCycleGroups) : "draft";
    const isDropDisabled = cycleStatus === "completed" || Boolean(cycle.archived_at);
    cycles.push({
      id: cycle.id,
      name: cycle.name,
      icon: <CycleGroupIcon cycleGroup={cycleStatus} className="h-3.5 w-3.5" />,
      payload: { cycle_id: cycle.id },
      isDropDisabled,
      dropErrorMessage: isDropDisabled ? "Work item cannot be moved to completed cycles" : undefined,
    });
  });
  cycles.push({
    id: "None",
    name: "None",
    icon: <CycleIcon className="h-3.5 w-3.5" />,
    payload: {},
  });
  return cycles;
};

const getModuleColumns = (moduleDetails: readonly IModule[] | undefined): IGroupByColumn[] | undefined => {
  if (!moduleDetails) return undefined;
  const modules: IGroupByColumn[] = [];
  moduleDetails?.map((module) => {
    modules.push({
      id: module.id,
      name: module.name,
      icon: <ModuleIcon className="h-3.5 w-3.5" />,
      payload: { module_ids: [module.id] },
      isDropDisabled: Boolean(module.archived_at),
      dropErrorMessage: module.archived_at ? "Work items cannot be moved to archived modules" : undefined,
    });
  });
  modules.push({
    id: "None",
    name: "None",
    icon: <ModuleIcon className="h-3.5 w-3.5" />,
    payload: {},
  });
  return modules;
};

const getStateColumns = ({
  projectId,
  states,
}: TGetColumns & { states?: readonly IState[] }): IGroupByColumn[] | undefined => {
  const _states = projectId ? states?.filter((state) => state.project_id === projectId) : states;
  if (!_states) return;
  // map project states to group by columns
  return _states.map((state) => ({
    id: state.id,
    name: state.name,
    icon: (
      <div className="size-4 rounded-full">
        <StateGroupIcon stateGroup={state.group} color={state.color} size={EIconSize.LG} percentage={state.order} />
      </div>
    ),
    payload: { state_id: state.id },
  }));
};

const getStateGroupColumns = (): IGroupByColumn[] => {
  const stateGroups = STATE_GROUPS;
  // map state groups to group by columns
  return Object.values(stateGroups).map((stateGroup) => ({
    id: stateGroup.key,
    name: stateGroup.label,
    icon: (
      <div className="size-4 rounded-full">
        <StateGroupIcon stateGroup={stateGroup.key} size={EIconSize.LG} />
      </div>
    ),
    payload: {},
  }));
};

const getPriorityColumns = (): IGroupByColumn[] => {
  const priorities = ISSUE_PRIORITIES;
  // map priorities to group by columns
  return priorities.map((priority) => ({
    id: priority.key,
    name: priority.title,
    icon: <PriorityIcon priority={priority?.key} />,
    payload: { priority: priority.key },
  }));
};

const getLabelsColumns = (nativeLabels: readonly IIssueLabel[] | undefined): IGroupByColumn[] => {
  const labels = [...(nativeLabels ?? []), { id: "None", name: "None", color: "#666" }];
  // map labels to group by columns
  return labels.map((label) => ({
    id: label.id,
    name: label.name,
    icon: (
      <div className="h-[12px] w-[12px] rounded-full" style={{ backgroundColor: label.color ? label.color : "#666" }} />
    ),
    payload: label?.id === "None" ? {} : { label_ids: [label.id] },
  }));
};

const getMemberColumns = (
  memberIds: readonly string[],
  membersById: ReadonlyMap<string, IWorkspaceMember> | undefined,
  kind: "assignees" | "created_by"
): IGroupByColumn[] => {
  const columns: IGroupByColumn[] = [];
  for (const memberId of memberIds) {
    const member = membersById?.get(memberId)?.member;
    if (!member) continue;
    columns.push({
      id: memberId,
      name: member.display_name || "",
      icon: <Avatar name={member.display_name} src={getFileURL(member.avatar_url ?? "")} size="md" />,
      payload: kind === "assignees" ? { assignee_ids: [memberId] } : {},
    });
  }
  if (kind === "assignees") columns.push({ id: "None", name: "None", icon: <Avatar size="md" />, payload: {} });
  return columns;
};

/**
 * This Method finds the DOM element with elementId, scrolls to it and highlights the issue block
 * @param elementId
 * @param shouldScrollIntoView
 */
export const highlightIssueOnDrop = (
  elementId: string | undefined,
  shouldScrollIntoView = true,
  shouldHighLightWithLine = false
) => {
  setTimeout(async () => {
    const sourceElementId = elementId ?? "";
    const sourceElement = document.getElementById(sourceElementId);
    sourceElement?.classList?.add(shouldHighLightWithLine ? HIGHLIGHT_WITH_LINE : HIGHLIGHT_CLASS);
    if (shouldScrollIntoView && sourceElement)
      await scrollIntoView(sourceElement, { behavior: "smooth", block: "center", duration: 1500 });
  }, 200);
};

/**
 * get Kanban Source data from Pragmatic Payload
 * @param payload
 * @returns
 */
export const getSourceFromDropPayload = (payload: IPragmaticDropPayload): GroupDropLocation | undefined => {
  const { location, source: sourceIssue } = payload;

  const sourceIssueData = sourceIssue.data;
  let sourceColumData;

  const sourceDropTargets = location?.initial?.dropTargets ?? [];
  for (const dropTarget of sourceDropTargets) {
    const dropTargetData = dropTarget?.data;

    if (!dropTargetData) continue;

    if (dropTargetData.type === "COLUMN") {
      sourceColumData = dropTargetData;
    }
  }

  if (sourceIssueData?.id === undefined || !sourceColumData?.groupId) return;

  return {
    groupId: sourceColumData.groupId as string,
    subGroupId: sourceColumData.subGroupId as string,
    columnId: sourceColumData.columnId as string,
    id: sourceIssueData.id as string,
  };
};

/**
 * get Destination Source data from Pragmatic Payload
 * @param payload
 * @returns
 */
export const getDestinationFromDropPayload = (payload: IPragmaticDropPayload): GroupDropLocation | undefined => {
  const { location } = payload;

  let destinationIssueData, destinationColumnData;

  const destDropTargets = location?.current?.dropTargets ?? [];

  for (const dropTarget of destDropTargets) {
    const dropTargetData = dropTarget?.data;

    if (!dropTargetData) continue;

    if (dropTargetData.type === "COLUMN" || dropTargetData.type === "DELETE") {
      destinationColumnData = dropTargetData;
    }

    if (dropTargetData.type === "ISSUE") {
      destinationIssueData = dropTargetData;
    }
  }

  if (!destinationColumnData?.groupId) return;

  // extract instruction from destination issue
  const extractedInstruction = destinationIssueData ? extractInstruction(destinationIssueData)?.type : "";

  return {
    groupId: destinationColumnData.groupId as string,
    subGroupId: destinationColumnData.subGroupId as string,
    columnId: destinationColumnData.columnId as string,
    id: destinationIssueData?.id as string | undefined,
    canAddIssueBelow: extractedInstruction === "reorder-below",
  };
};

/**
 * Returns Sort order of the issue block at the position of drop
 * @param destinationIssues
 * @param destinationIssueId
 * @param getIssueById
 * @returns
 */
const handleSortOrder = (
  destinationIssues: string[],
  destinationIssueId: string | undefined,
  getIssueById: (issueId: string) => TIssue | undefined,
  shouldAddIssueAtTop = false,
  canAddIssueBelow = false
) => {
  const sortOrderDefaultValue = 65535;
  let currentIssueState = {};

  let destinationIndex = destinationIssueId
    ? destinationIssues.indexOf(destinationIssueId)
    : shouldAddIssueAtTop
      ? 0
      : destinationIssues.length;

  const isDestinationLastChild = destinationIndex === destinationIssues.length - 1;

  // if issue can be added below and if the destination issue is the last child, then add to the end of the list
  if (canAddIssueBelow && isDestinationLastChild) {
    destinationIndex = destinationIssues.length;
  }

  if (destinationIssues && destinationIssues.length > 0) {
    if (destinationIndex === 0) {
      const destinationIssueId = destinationIssues[0];
      const destinationIssue = getIssueById(destinationIssueId);
      if (!destinationIssue) return currentIssueState;

      currentIssueState = {
        ...currentIssueState,
        sort_order: destinationIssue.sort_order - sortOrderDefaultValue,
      };
    } else if (destinationIndex === destinationIssues.length) {
      const destinationIssueId = destinationIssues[destinationIssues.length - 1];
      const destinationIssue = getIssueById(destinationIssueId);
      if (!destinationIssue) return currentIssueState;

      currentIssueState = {
        ...currentIssueState,
        sort_order: destinationIssue.sort_order + sortOrderDefaultValue,
      };
    } else {
      const destinationTopIssueId = destinationIssues[destinationIndex - 1];
      const destinationBottomIssueId = destinationIssues[destinationIndex];

      const destinationTopIssue = getIssueById(destinationTopIssueId);
      const destinationBottomIssue = getIssueById(destinationBottomIssueId);
      if (!destinationTopIssue || !destinationBottomIssue) return currentIssueState;

      currentIssueState = {
        ...currentIssueState,
        sort_order: (destinationTopIssue.sort_order + destinationBottomIssue.sort_order) / 2,
      };
    }
  } else {
    currentIssueState = {
      ...currentIssueState,
      sort_order: sortOrderDefaultValue,
    };
  }

  return currentIssueState;
};

export const getIssueBlockId = (issueId: string | undefined, groupId: string | undefined, subGroupId?: string) =>
  `issue_${issueId}_${groupId}_${subGroupId}`;

/**
 * returns empty Array if groupId is None
 * @param groupId
 * @returns
 */
const getGroupId = (groupId: string) => {
  if (groupId === "None") return [];
  return [groupId];
};

function moveGroupValue(value: TIssue[keyof TIssue], sourceId: string, destinationId: string) {
  if (!Array.isArray(value)) return destinationId === "None" ? null : destinationId;

  const nextValues = value.filter((entry): entry is string => typeof entry === "string" && entry !== sourceId);
  if (destinationId !== "None" && !nextValues.includes(destinationId)) nextValues.push(destinationId);
  return nextValues;
}

export const handleGroupDragDrop = async (
  source: GroupDropLocation,
  destination: GroupDropLocation,
  getIssueById: (issueId: string) => TIssue | undefined,
  getIssueIds: (groupId?: string, subGroupId?: string) => string[] | undefined,
  updateIssueOnDrop: (projectId: string, issueId: string, data: Partial<TIssue>, issueUpdates: IssueUpdates) => void,
  groupBy: TIssueGroupByOptions | undefined,
  subGroupBy: TIssueGroupByOptions | undefined,
  shouldAddIssueAtTop = false
) => {
  if (!source.id || (subGroupBy && (!source.subGroupId || !destination.subGroupId))) return;

  let updatedIssue: Partial<TIssue> = {};
  const issueUpdates: IssueUpdates = {};
  const destinationIssues = getIssueIds(destination.groupId, destination.subGroupId) ?? [];

  const sourceIssue = getIssueById(source.id);

  if (!sourceIssue) return;

  updatedIssue = {
    id: sourceIssue.id,
    project_id: sourceIssue.project_id,
  };

  // for both horizontal and vertical dnd
  updatedIssue = {
    ...updatedIssue,
    ...handleSortOrder(
      destinationIssues,
      destination.id,
      getIssueById,
      shouldAddIssueAtTop,
      !!destination.canAddIssueBelow
    ),
  };

  // update updatedIssue values based on the source and destination groupIds
  if (source.groupId && destination.groupId && source.groupId !== destination.groupId && groupBy) {
    const groupKey = ISSUE_FILTER_DEFAULT_DATA[groupBy];
    const groupValue = moveGroupValue(sourceIssue[groupKey], source.groupId, destination.groupId);

    // keep track of updates on what was added and what was removed
    issueUpdates[groupKey] = { ADD: getGroupId(destination.groupId), REMOVE: getGroupId(source.groupId) };
    updatedIssue = { ...updatedIssue, [groupKey]: groupValue };
  }

  // do the same for subgroup
  // update updatedIssue values based on the source and destination subGroupIds
  if (subGroupBy && source.subGroupId && destination.subGroupId && source.subGroupId !== destination.subGroupId) {
    const subGroupKey = ISSUE_FILTER_DEFAULT_DATA[subGroupBy];
    const subGroupValue = moveGroupValue(sourceIssue[subGroupKey], source.subGroupId, destination.subGroupId);

    // keep track of updates on what was added and what was removed
    issueUpdates[subGroupKey] = { ADD: getGroupId(destination.subGroupId), REMOVE: getGroupId(source.subGroupId) };
    updatedIssue = { ...updatedIssue, [subGroupKey]: subGroupValue };
  }

  if (updatedIssue && sourceIssue?.project_id) {
    return await updateIssueOnDrop(sourceIssue?.project_id, sourceIssue.id, updatedIssue, issueUpdates);
  }
};

/**
 * method that removes Null or undefined Keys from object
 * @param obj
 * @returns
 */
export const removeNillKeys = <T,>(obj: T) =>
  Object.fromEntries(Object.entries(obj ?? {}).filter(([key, value]) => key && !isNil(value)));

/**
 * This Method returns if the issue is new or not
 * @param issue
 * @returns
 */
export const isIssueNew = (issue: TIssue) => {
  const createdDate = new Date(issue.created_at);
  const currentDate = new Date();
  const diff = currentDate.getTime() - createdDate.getTime();
  return diff < 30000;
};

/**
 * Returns approximate height of Kanban card based on display properties
 * @param displayProperties
 * @returns
 */
export function getApproximateCardHeight(displayProperties: IIssueDisplayProperties | undefined) {
  if (!displayProperties) return 100;

  // default card height
  let cardHeight = 46;

  const clonedProperties = clone(displayProperties);

  // The key and type share one header row.
  if ([clonedProperties.key, clonedProperties.issue_type].some(Boolean)) {
    cardHeight += 24;
  }

  // Ignore smaller dimension properties
  const ignoredProperties: (keyof IIssueDisplayProperties)[] = [
    "key",
    "issue_type",
    "sub_issue_count",
    "link",
    "attachment_count",
    "created_on",
    "updated_on",
  ];

  ignoredProperties.forEach((key: keyof IIssueDisplayProperties) => {
    delete clonedProperties[key];
  });

  let propertyCount = 0;

  // count the remaining properties
  (Object.keys(clonedProperties) as (keyof IIssueDisplayProperties)[]).forEach((key: keyof IIssueDisplayProperties) => {
    if (!key.startsWith("customproperty_") && clonedProperties[key]) {
      propertyCount++;
    }
  });

  // based on property count, approximate the height of each card
  if (propertyCount > 3) {
    cardHeight += 60;
  } else if (propertyCount > 0) {
    cardHeight += 32;
  }

  return cardHeight;
}

/**
 * This Method is used to get Block view details, that returns block style and tooltip message
 * @param block
 * @param backgroundColor
 * @returns
 */
export const getBlockViewDetails = (
  block: { start_date: string | undefined | null; target_date: string | undefined | null } | undefined | null,
  backgroundColor: string
) => {
  const isBlockVisibleOnChart = block?.start_date || block?.target_date;
  const isBlockComplete = block?.start_date && block?.target_date;

  let message;
  const blockStyle: CSSProperties = {
    backgroundColor,
  };

  if (isBlockVisibleOnChart && !isBlockComplete) {
    if (block?.start_date) {
      message = `From ${renderFormattedDate(block.start_date)}`;
      blockStyle.maskImage = `linear-gradient(to right, ${backgroundColor} 50%, transparent 95%)`;
    } else if (block?.target_date) {
      message = `Till ${renderFormattedDate(block.target_date)}`;
      blockStyle.maskImage = `linear-gradient(to left, ${backgroundColor} 50%, transparent 95%)`;
    }
  } else if (isBlockComplete) {
    message = `${renderFormattedDate(block?.start_date)} to ${renderFormattedDate(block?.target_date)}`;
  }

  return {
    message,
    blockStyle,
  };
};

/**
 * This method returns the icon for Spreadsheet column headers
 * @param iconKey
 */
export function SpreadSheetPropertyIcon(props: ISvgIcons & { iconKey: string }) {
  const { iconKey, ...iconProps } = props;
  const Icon = SpreadSheetPropertyIconMap[iconKey];
  if (!Icon) return null;
  return <Icon {...iconProps} />;
}

/**
 * This method returns if the filters are applied
 * @param filters
 * @returns
 */
export const isDisplayFiltersApplied = (filters: Partial<IIssueFilters>): boolean => {
  const isDisplayPropertiesApplied = Object.keys(DEFAULT_SUB_WORK_ITEM_DISPLAY_PROPERTIES).some(
    (key) => !filters.displayProperties?.[key as keyof IIssueDisplayProperties]
  );

  const isDisplayFiltersApplied = Object.keys(filters.displayFilters ?? {}).some((key) => {
    const value = filters.displayFilters?.[key as keyof IIssueDisplayFilterOptions];
    if (!value) return false;
    // -create_at is the default order
    if (key === "order_by") {
      return value !== "-created_at";
    }
    return true;
  });

  return isDisplayPropertiesApplied || isDisplayFiltersApplied;
};

/**
 * This method returns if the filters are applied
 * @param filters
 * @returns
 */
export const isFiltersApplied = (filters: IIssueFilterOptions): boolean =>
  Object.values(filters).some((value) => {
    if (Array.isArray(value)) return value.length > 0;
    return value !== undefined && value !== null && value !== "";
  });

/**
 * Calculates the minimum width needed for issue identifiers in list layouts
 * @param projectIdentifierLength - Length of the project identifier (e.g., "PROJ" = 4)
 * @param maxSequenceId - Maximum sequence ID in the project (e.g., 1234)
 * @returns Width in pixels needed to display the identifier
 *
 * @example
 * // For "PROJ-1234"
 * calculateIdentifierWidth(4, 1234) // Returns width for "PROJ" + "-" + "1234"
 */
export const calculateIdentifierWidth = (projectIdentifierLength: number, maxSequenceId: number): number => {
  const sequenceDigits = Math.max(1, Math.floor(Math.log10(maxSequenceId)) + 1);
  return projectIdentifierLength * 7 + 7 + sequenceDigits * 7; // project identifier chars + dash + sequence digits
};

const getTeamProjectColumns = (): IGroupByColumn[] | undefined => undefined;

const SpreadSheetPropertyIconMap: Record<string, FC<ISvgIcons>> = {
  MembersPropertyIcon: MembersPropertyIcon,
  CalenderDays: CalendarDays,
  DueDatePropertyIcon: DueDatePropertyIcon,
  EstimatePropertyIcon: EstimatePropertyIcon,
  LabelPropertyIcon: LabelPropertyIcon,
  ModuleIcon: ModuleIcon,
  ContrastIcon: CycleIcon,
  PriorityPropertyIcon: PriorityPropertyIcon,
  StartDatePropertyIcon: StartDatePropertyIcon,
  StatePropertyIcon: StatePropertyIcon,
  Link2: LinkIcon,
  Paperclip: Paperclip,
  LayersIcon: LayersIcon,
};

export const SPREADSHEET_COLUMNS: { [key in TIssueStandardDisplayProperty]?: TSpreadsheetColumn } = {
  assignee: SpreadsheetAssigneeColumn,
  created_on: SpreadsheetCreatedOnColumn,
  due_date: SpreadsheetDueDateColumn,
  estimate: SpreadsheetEstimateColumn,
  labels: SpreadsheetLabelColumn,
  modules: SpreadsheetModuleColumn,
  cycle: SpreadsheetCycleColumn,
  link: SpreadsheetLinkColumn,
  priority: SpreadsheetPriorityColumn,
  start_date: SpreadsheetStartDateColumn,
  state: SpreadsheetStateColumn,
  sub_issue_count: SpreadsheetSubIssueColumn,
  updated_on: SpreadsheetUpdatedOnColumn,
  attachment_count: SpreadsheetAttachmentColumn,
};

export const useGroupByOptions = (
  options: TIssueGroupByOptions[]
): {
  key: TIssueGroupByOptions;
  titleTranslationKey: string;
}[] => {
  const groupByOptions = ISSUE_GROUP_BY_OPTIONS.filter((option) => options.includes(option.key));
  return groupByOptions;
};
