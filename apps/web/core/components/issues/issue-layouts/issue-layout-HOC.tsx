/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AlertTriangle } from "lucide-react";
import { useTranslation } from "@plane/i18n";
// plane imports
import { EIssueLayoutTypes } from "@plane/types";
import { Button } from "@plane/propel/button";
// components
import { LayoutErrorBoundary } from "@/components/common/layout-error-boundary";
import { CalendarLayoutLoader } from "@/components/ui/loader/layouts/calendar-layout-loader";
import { GanttLayoutLoader } from "@/components/ui/loader/layouts/gantt-layout-loader";
import { KanbanLayoutLoader } from "@/components/ui/loader/layouts/kanban-layout-loader";
import { ListLayoutLoader } from "@/components/ui/loader/layouts/list-layout-loader";
import { SpreadsheetLayoutLoader } from "@/components/ui/loader/layouts/spreadsheet-layout-loader";
// hooks
import { useIssueListType } from "@/hooks/use-issue-list-type";
// local imports
import { IssueLayoutEmptyState } from "./empty-states";

function ActiveLoader(props: { layout: EIssueLayoutTypes }) {
  const { layout } = props;
  switch (layout) {
    case EIssueLayoutTypes.LIST:
      return <ListLayoutLoader />;
    case EIssueLayoutTypes.KANBAN:
      return <KanbanLayoutLoader />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <SpreadsheetLayoutLoader />;
    case EIssueLayoutTypes.CALENDAR:
      return <CalendarLayoutLoader />;
    case EIssueLayoutTypes.GANTT:
      return <GanttLayoutLoader />;
    default:
      return null;
  }
}

interface Props {
  allowEmptyGroups?: boolean;
  children: string | React.ReactNode | React.ReactNode[];
  layout: EIssueLayoutTypes;
  readState: {
    isPending: boolean;
    isError: boolean;
    issueCount: number;
    retry: () => Promise<unknown>;
  };
}

export function IssueLayoutHOC(props: Props) {
  const { layout, readState } = props;
  const retry = readState.retry;
  const { t } = useTranslation();

  const storeType = useIssueListType();
  const issueCount = readState.issueCount;

  if (readState.isError) {
    return (
      <div role="alert" className="flex h-full w-full flex-col items-center justify-center gap-3 text-center">
        <AlertTriangle className="size-8 text-tertiary" />
        <p className="text-14 text-secondary">{t("something_went_wrong")}</p>
        {retry && (
          <Button variant="secondary" size="sm" onClick={() => void retry()}>
            {t("common.retry")}
          </Button>
        )}
      </div>
    );
  }

  if (readState.isPending)
    return (
      <output aria-label="Loading work items">
        <ActiveLoader layout={layout} />
      </output>
    );

  if (issueCount === 0 && layout !== EIssueLayoutTypes.CALENDAR && !props.allowEmptyGroups) {
    return <IssueLayoutEmptyState storeType={storeType} />;
  }

  return <LayoutErrorBoundary key={layout}>{props.children}</LayoutErrorBoundary>;
}
