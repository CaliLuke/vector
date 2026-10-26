import type { WorkItemPropertyIdentity } from "@/services/work-item-types.service";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { MutableRefObject } from "react";
import { useCallback, useEffect, useRef, useState } from "react";

// plane imports
import type {
  TIssueStandardDisplayProperty,
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  TIssue,
} from "@plane/types";
// components
import { SpreadsheetIssueRowLoader } from "@/components/ui/loader/layouts/spreadsheet-layout-loader";
// hooks
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
import { useIssueListView } from "@/providers/issue-list-view";
import type { TSelectionHelper } from "@/hooks/use-multiple-select";
import { useTableKeyboardNavigation } from "@/hooks/use-table-keyboard-navigation";
// local imports
import type { TRenderQuickActions } from "../list/list-view-types";
import { shouldRenderColumn } from "@/helpers/issue-filter.helper";
import { SpreadsheetIssueRow } from "./issue-row";
import { SpreadsheetHeader } from "./spreadsheet-header";

type Props = {
  onEditProperties: (issue: TIssue) => void;
  customColumns: WorkItemPropertyIdentity[];
  displayProperties: IIssueDisplayProperties;
  displayFilters: IIssueDisplayFilterOptions;
  handleDisplayFilterUpdate: (data: Partial<IIssueDisplayFilterOptions>) => void;
  issueIds: string[];
  isEstimateEnabled: boolean;
  quickActions: TRenderQuickActions;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  canEditProperties: (projectId: string | undefined) => boolean;
  portalElement: React.MutableRefObject<HTMLDivElement | null>;
  containerRef: MutableRefObject<HTMLTableElement | null>;
  canLoadMoreIssues: boolean;
  loadMoreIssues: () => void;
  spreadsheetColumnsList: TIssueStandardDisplayProperty[];
  propertyRevision?: number;
  selectionHelpers: TSelectionHelper;
  isEpic?: boolean;
};

export const SpreadsheetTable = function SpreadsheetTable(props: Props) {
  const {
    onEditProperties,
    customColumns,
    displayProperties,
    displayFilters,
    handleDisplayFilterUpdate,
    issueIds,
    isEstimateEnabled,
    portalElement,
    quickActions,
    updateIssue,
    canEditProperties,
    canLoadMoreIssues,
    containerRef,
    loadMoreIssues,
    spreadsheetColumnsList,
    propertyRevision,
    selectionHelpers,
    isEpic = false,
  } = props;

  // states
  const isScrolled = useRef(false);
  const [intersectionElement, setIntersectionElement] = useState<HTMLTableSectionElement | null>(null);

  const { isLoadingNextPage } = useIssueListView();

  const handleScroll = useCallback(() => {
    if (!containerRef.current) return;
    const scrollLeft = containerRef.current.scrollLeft;

    const columnShadow = "8px 22px 22px 10px rgba(0, 0, 0, 0.05)"; // shadow for regular columns
    const headerShadow = "8px -22px 22px 10px rgba(0, 0, 0, 0.05)"; // shadow for headers

    //The shadow styles are added this way to avoid re-render of all the rows of table, which could be costly
    if (scrollLeft > 0 !== isScrolled.current) {
      const firstColumns = containerRef.current.querySelectorAll("table tr td:first-child, th:first-child");

      for (let i = 0; i < firstColumns.length; i++) {
        const shadow = i === 0 ? headerShadow : columnShadow;
        if (scrollLeft > 0) {
          (firstColumns[i] as HTMLElement).style.boxShadow = shadow;
        } else {
          (firstColumns[i] as HTMLElement).style.boxShadow = "none";
        }
      }
      isScrolled.current = scrollLeft > 0;
    }
  }, [containerRef]);

  useEffect(() => {
    const currentContainerRef = containerRef.current;

    if (currentContainerRef) currentContainerRef.addEventListener("scroll", handleScroll);

    return () => {
      if (currentContainerRef) currentContainerRef.removeEventListener("scroll", handleScroll);
    };
  }, [handleScroll, containerRef]);

  const isPaginating = isLoadingNextPage();

  useIntersectionObserver(containerRef, isPaginating ? null : intersectionElement, loadMoreIssues, `100% 0% 100% 0%`);

  const handleKeyBoardNavigation = useTableKeyboardNavigation();

  const displayPropertiesCount =
    spreadsheetColumnsList.filter(
      (property) => displayProperties[property] && shouldRenderColumn(property, isEstimateEnabled)
    ).length + customColumns.length;

  return (
    <table className="w-full overflow-y-auto bg-surface-1" onKeyDown={handleKeyBoardNavigation}>
      <SpreadsheetHeader
        customColumns={customColumns}
        displayProperties={displayProperties}
        displayFilters={displayFilters}
        handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        canEditProperties={canEditProperties}
        isEstimateEnabled={isEstimateEnabled}
        spreadsheetColumnsList={spreadsheetColumnsList}
        selectionHelpers={selectionHelpers}
        isEpic={isEpic}
      />
      <tbody>
        {issueIds.map((id) => (
          <SpreadsheetIssueRow
            key={id}
            onEditProperties={onEditProperties}
            issueId={id}
            customColumns={customColumns}
            displayProperties={displayProperties}
            quickActions={quickActions}
            canEditProperties={canEditProperties}
            nestingLevel={0}
            isEstimateEnabled={isEstimateEnabled}
            updateIssue={updateIssue}
            portalElement={portalElement}
            containerRef={containerRef}
            isScrolled={isScrolled}
            spreadsheetColumnsList={spreadsheetColumnsList}
            propertyRevision={propertyRevision}
            selectionHelpers={selectionHelpers}
            isEpic={isEpic}
          />
        ))}
      </tbody>
      {canLoadMoreIssues && (
        <tfoot ref={setIntersectionElement}>
          {Array.from({ length: 3 }).map((_, index) => (
            <SpreadsheetIssueRowLoader key={index} columnCount={displayPropertiesCount} />
          ))}
        </tfoot>
      )}
    </table>
  );
};
