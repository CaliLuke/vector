import type { WorkItemPropertyIdentity } from "@/services/work-item-types.service";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { useContext, useMemo, useRef, useState } from "react";

import { MoreHorizontal } from "lucide-react";
import { SPREADSHEET_SELECT_GROUP } from "@plane/constants";
// plane helpers
import { useOutsideClickDetector } from "@plane/hooks";
import { ChevronRightIcon } from "@plane/propel/icons";
// types
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssueStandardDisplayProperty, IIssueDisplayProperties, TIssue } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// ui
import { ControlLink, Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { generateWorkItemLink } from "@plane/utils/work-item";
// components
import { MultipleSelectEntityAction } from "@/components/core/multiple-select";
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";
import { useIssueListView } from "@/providers/issue-list-view";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import type { TSelectionHelper } from "@/hooks/use-multiple-select";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local components
import type { TRenderQuickActions } from "../list/list-view-types";
import { isIssueNew } from "../utils";
import { useSubItemExpansion } from "./use-sub-item-expansion";
import { TablePropertyValuesContext } from "@/hooks/use-work-item-table-values";
import { useWorkItemTableValues } from "@/hooks/use-work-item-table-values";
import { PropertyColumn } from "./property-column";
import { IssueColumn } from "./issue-column";

interface Props {
  onEditProperties: (issue: TIssue) => void;
  customColumns: WorkItemPropertyIdentity[];
  displayProperties: IIssueDisplayProperties;
  isEstimateEnabled: boolean;
  quickActions: TRenderQuickActions;
  canEditProperties: (projectId: string | undefined) => boolean;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  portalElement: React.MutableRefObject<HTMLDivElement | null>;
  nestingLevel: number;
  issueId: string;
  issue?: TIssue;
  propertyRevision?: number;
  isScrolled: MutableRefObject<boolean>;
  containerRef: MutableRefObject<HTMLTableElement | null>;
  spreadsheetColumnsList: TIssueStandardDisplayProperty[];
  spacingLeft?: number;
  selectionHelpers: TSelectionHelper;
  shouldRenderByDefault?: boolean;
  isEpic?: boolean;
}

export function SpreadsheetIssueRow(props: Props) {
  const {
    onEditProperties,
    customColumns,
    displayProperties,
    issueId,
    issue: issueProp,
    propertyRevision,
    isEstimateEnabled,
    nestingLevel,
    portalElement,
    updateIssue,
    quickActions,
    canEditProperties,
    isScrolled,
    containerRef,
    spreadsheetColumnsList,
    spacingLeft = 6,
    selectionHelpers,
    shouldRenderByDefault,
    isEpic = false,
  } = props;
  // states
  const [isExpanded, setExpanded] = useState<boolean>(false);
  // store hooks
  const { issuesById } = useIssueListView();

  // derived values
  const issue = issueProp ?? issuesById[issueId];
  const { workspaceSlug } = useParams({ strict: false });
  const childQuery = useWorkItemSubItems(
    workspaceSlug?.toString(),
    issue?.project_id ?? undefined,
    issueId,
    isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES,
    isExpanded
  );
  const subItems = childQuery.subItems;
  const inheritedPropertyValues = useContext(TablePropertyValuesContext);
  const childPropertyReads = useWorkItemTableValues(
    workspaceSlug?.toString() ?? "",
    subItems ?? [],
    propertyRevision,
    isExpanded && customColumns.length > 0,
    customColumns.some((field) => field.kind === "Formula")
  );
  const propertyValues = useMemo(
    () => ({ ...inheritedPropertyValues, ...childPropertyReads.values }),
    [inheritedPropertyValues, childPropertyReads.values]
  );
  const isIssueSelected = selectionHelpers.getIsEntitySelected(issueId);
  const isIssueActive = selectionHelpers.getIsEntityActive(issueId);

  if (!issue) return null;

  return (
    <TablePropertyValuesContext.Provider value={propertyValues}>
      {/* first column/ issue name and key column */}
      <RenderIfVisible
        as="tr"
        root={containerRef}
        placeholderChildren={
          <td
            colSpan={100}
            className="border-[0.5px] border-transparent border-b-subtle-1"
            style={{ height: "calc(2.75rem - 1px)" }}
          />
        }
        classNames={cn("bg-surface-1 transition-[background-color]", {
          "group selected-issue-row": isIssueSelected,
          "border-[0.5px] border-strong-1": isIssueActive,
        })}
        verticalOffset={100}
        shouldRecordHeights={false}
        defaultValue={shouldRenderByDefault || isIssueNew(issue)}
      >
        <IssueRowDetails
          onEditProperties={onEditProperties}
          issueId={issueId}
          issue={issue}
          propertyRevision={propertyRevision}
          customColumns={customColumns}
          displayProperties={displayProperties}
          quickActions={quickActions}
          canEditProperties={canEditProperties}
          nestingLevel={nestingLevel}
          spacingLeft={spacingLeft}
          isEstimateEnabled={isEstimateEnabled}
          updateIssue={updateIssue}
          portalElement={portalElement}
          isScrolled={isScrolled}
          isExpanded={isExpanded}
          setExpanded={setExpanded}
          spreadsheetColumnsList={spreadsheetColumnsList}
          selectionHelpers={selectionHelpers}
          isEpic={isEpic}
        />
      </RenderIfVisible>

      <ExpandedSpreadsheetSubItems
        rowProps={props}
        subItems={subItems}
        isExpanded={isExpanded}
        isLoaded={childQuery.isSuccess}
        isEpic={isEpic}
      />
    </TablePropertyValuesContext.Provider>
  );
}

function ExpandedSpreadsheetSubItems({
  rowProps,
  subItems,
  isExpanded,
  isLoaded,
  isEpic,
}: {
  rowProps: Props;
  subItems: TIssue[] | undefined;
  isExpanded: boolean;
  isLoaded: boolean;
  isEpic: boolean;
}) {
  if (!isExpanded || isEpic || !isLoaded || !subItems) return null;

  return subItems.map((subItem) => (
    <SpreadsheetIssueRow
      key={subItem.id}
      {...rowProps}
      issueId={subItem.id}
      issue={subItem}
      nestingLevel={rowProps.nestingLevel + 1}
      spacingLeft={(rowProps.spacingLeft ?? 6) + 12}
      shouldRenderByDefault={isExpanded}
    />
  ));
}

interface IssueRowDetailsProps extends Omit<Props, "containerRef" | "shouldRenderByDefault"> {
  isExpanded: boolean;
  setExpanded: Dispatch<SetStateAction<boolean>>;
}

function IssueRowDetails(props: IssueRowDetailsProps) {
  const {
    onEditProperties,
    customColumns,
    displayProperties,
    issueId,
    issue,
    isEstimateEnabled,
    nestingLevel,
    portalElement,
    updateIssue,
    quickActions,
    canEditProperties,
    isScrolled,
    isExpanded,
    setExpanded,
    spreadsheetColumnsList,
    spacingLeft = 6,
    selectionHelpers,
    isEpic = false,
  } = props;
  // states
  const [isMenuActive, setIsMenuActive] = useState(false);
  // refs
  const cellRef = useRef(null);
  const menuActionRef = useRef<HTMLDivElement | null>(null);
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const router = useRouter();
  const showIdentity = Boolean(displayProperties.key || displayProperties.issue_type);
  // hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const { getIsIssuePeeked, peekIssue, setPeekIssue } = useIssueDetailUiState(
    isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );
  const { isMobile } = usePlatformOS();

  // handlers
  const handleIssuePeekOverview = (item: TIssue) => {
    if (!workspaceSlug || !item.project_id || getIsIssuePeeked(item.id) || item.tempId) return;
    const itemProjectIdentifier = projectRows?.find((project) => project.id === item.project_id)?.identifier ?? "";
    const itemLink = generateWorkItemLink({
      workspaceSlug,
      projectId: item.project_id,
      issueId: item.id,
      projectIdentifier: itemProjectIdentifier,
      sequenceId: item.sequence_id,
      isEpic,
    });
    if (isMobile) void router.navigate({ href: itemLink });
    else setPeekIssue({ workspaceSlug, projectId: item.project_id, issueId: item.id, nestingLevel });
  };

  const issueDetail = issue;
  const expansion = useSubItemExpansion({
    workspaceSlug,
    issue: issueDetail,
    nestingLevel,
    expanded: isExpanded,
    setExpanded,
    onOpen: handleIssuePeekOverview,
  });

  const subIssueIndentation = `${spacingLeft}px`;

  useOutsideClickDetector(menuActionRef, () => setIsMenuActive(false));

  const customActionButton = (
    <div
      ref={menuActionRef}
      className={`flex h-full w-full cursor-pointer items-center rounded-sm p-1 text-placeholder hover:bg-layer-1 ${
        isMenuActive ? "bg-layer-1 text-primary" : "text-secondary"
      }`}
      onClick={() => setIsMenuActive(!isMenuActive)}
    >
      <MoreHorizontal className="h-3.5 w-3.5" />
    </div>
  );
  if (!issueDetail) return null;

  const disableUserActions = !canEditProperties(issueDetail.project_id ?? undefined);
  const subIssuesCount = issueDetail?.sub_issues_count ?? 0;
  const isIssueSelected = selectionHelpers.getIsEntitySelected(issueDetail.id);
  const projectIdentifier = projectRows?.find((project) => project.id === issueDetail.project_id)?.identifier ?? "";

  const canSelectIssues = !disableUserActions && !selectionHelpers.isSelectionDisabled;

  const workItemLink = generateWorkItemLink({
    workspaceSlug: workspaceSlug?.toString(),
    projectId: issueDetail?.project_id,
    issueId,
    projectIdentifier,
    sequenceId: issueDetail?.sequence_id,
    isEpic,
  });

  return (
    <>
      {/* Single sticky column containing both identifier and workitem */}
      <td
        id={`issue-${issueId}`}
        ref={cellRef}
        tabIndex={0}
        className="group/list-block relative left-0 z-10 max-w-lg bg-surface-1 md:sticky"
      >
        <ControlLink
          href={workItemLink}
          onClick={() => handleIssuePeekOverview(issueDetail)}
          className="outline-none"
          disabled={!!issueDetail?.tempId}
        >
          <Row
            className={cn(
              "group clickable z-10 flex h-11 w-full cursor-pointer items-center border-r-[0.5px] border-subtle-1 bg-transparent text-13 group-[.selected-issue-row]:bg-accent-primary/5 after:absolute group-[.selected-issue-row]:hover:bg-accent-primary/10",
              {
                "border-b-[0.5px]": !getIsIssuePeeked(issueDetail.id),
                "border border-accent-strong hover:border-accent-strong":
                  getIsIssuePeeked(issueDetail.id) && nestingLevel === peekIssue?.nestingLevel,
                "shadow-[8px_22px_22px_10px_rgba(0,0,0,0.05)]": isScrolled.current,
              }
            )}
          >
            <SpreadsheetIssueNavigation
              issue={issueDetail}
              workspaceSlug={workspaceSlug}
              displayProperties={displayProperties}
              showIdentity={showIdentity}
              projectId={projectId}
              canSelectIssues={canSelectIssues}
              isIssueSelected={isIssueSelected}
              selectionHelpers={selectionHelpers}
              nestingLevel={nestingLevel}
              subIssueIndentation={subIssueIndentation}
              subIssuesCount={subIssuesCount}
              isEpic={isEpic}
              expansion={expansion}
              isExpanded={isExpanded}
            >
              <div className="my-auto flex h-full w-full items-center justify-between gap-2 truncate">
                <div className="line-clamp-1 w-full text-14 text-primary">
                  <div className="w-full overflow-hidden">
                    <Tooltip tooltipContent={issueDetail.name} isMobile={isMobile}>
                      <div
                        className="h-full w-full cursor-pointer truncate pr-4 text-left text-13 text-primary focus:outline-none"
                        tabIndex={-1}
                      >
                        {issueDetail.name}
                      </div>
                    </Tooltip>
                  </div>
                </div>
                <div
                  className={`opacity-0 transition-opacity group-hover:opacity-100 ${isMenuActive ? "!opacity-100" : ""}`}
                  onClick={(e) => e.stopPropagation()}
                >
                  {quickActions({
                    issue: issueDetail,
                    parentRef: cellRef,
                    customActionButton,
                    portalElement: portalElement.current,
                  })}
                </div>
              </div>
            </SpreadsheetIssueNavigation>
          </Row>
        </ControlLink>
      </td>
      {/* Rest of the columns */}
      {spreadsheetColumnsList.map((property) => (
        <IssueColumn
          key={property}
          displayProperties={displayProperties}
          issueDetail={issueDetail}
          disableUserActions={disableUserActions}
          property={property}
          updateIssue={updateIssue}
          isEstimateEnabled={isEstimateEnabled}
        />
      ))}
      {customColumns.map((field) => (
        <PropertyColumn
          key={field.key}
          field={field}
          issue={issueDetail}
          disabled={disableUserActions}
          onEdit={onEditProperties}
        />
      ))}
    </>
  );
}

function SpreadsheetIssueNavigation({
  issue,
  workspaceSlug,
  displayProperties,
  showIdentity,
  projectId,
  canSelectIssues,
  isIssueSelected,
  selectionHelpers,
  nestingLevel,
  subIssueIndentation,
  subIssuesCount,
  isEpic,
  expansion,
  isExpanded,
  children,
}: {
  issue: TIssue;
  workspaceSlug: string | undefined;
  displayProperties: IIssueDisplayProperties;
  showIdentity: boolean;
  projectId: string | undefined;
  canSelectIssues: boolean;
  isIssueSelected: boolean;
  selectionHelpers: TSelectionHelper;
  nestingLevel: number;
  subIssueIndentation: string;
  subIssuesCount: number;
  isEpic: boolean;
  expansion: ReturnType<typeof useSubItemExpansion>;
  isExpanded: boolean;
  children: React.ReactNode;
}) {
  return (
    <>
      {showIdentity && (
        <div className="flex h-full min-w-24 flex-shrink-0 items-center">
          <div className="relative flex cursor-pointer items-center text-11 hover:text-primary">
            {issue.project_id && (
              <IssueIdentifier
                workspaceSlug={workspaceSlug}
                issueId={issue.id}
                projectId={issue.project_id}
                size="xs"
                variant="tertiary"
                displayProperties={displayProperties}
              />
            )}
          </div>
        </div>
      )}
      <div
        className={cn("flex flex-grow items-center gap-0.5 py-2", {
          "min-w-[360px]": !showIdentity,
          "min-w-60": showIdentity,
        })}
      >
        {projectId && canSelectIssues && (
          <Tooltip
            tooltipContent={
              <>
                Only work items within the current
                <br />
                project can be selected.
              </>
            }
            disabled={issue.project_id === projectId}
          >
            <div className="absolute left-1 mr-1 grid w-3.5 flex-shrink-0 place-items-center">
              <MultipleSelectEntityAction
                className={cn(
                  "pointer-events-none opacity-0 transition-opacity group-hover/list-block:pointer-events-auto group-hover/list-block:opacity-100",
                  { "pointer-events-auto opacity-100": isIssueSelected }
                )}
                groupId={SPREADSHEET_SELECT_GROUP}
                id={issue.id}
                selectionHelpers={selectionHelpers}
                disabled={issue.project_id !== projectId}
              />
            </div>
          </Tooltip>
        )}
        {nestingLevel !== 0 && <div style={{ width: subIssueIndentation }} />}
        <div className="grid size-4 place-items-center">
          {subIssuesCount > 0 && !isEpic && (
            <button
              type="button"
              className="grid size-4 place-items-center rounded-xs text-placeholder hover:text-tertiary"
              aria-label={expansion.label}
              title={expansion.title}
              onClick={expansion.onClick}
            >
              <ChevronRightIcon className={cn("size-4", { "rotate-90": isExpanded })} strokeWidth={2.5} />
            </button>
          )}
        </div>
        {children}
      </div>
    </>
  );
}
