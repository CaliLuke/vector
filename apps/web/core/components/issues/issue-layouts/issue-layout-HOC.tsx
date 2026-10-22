/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
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
import { useIssues } from "@/hooks/store/use-issues";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
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
  children: string | React.ReactNode | React.ReactNode[];
  layout: EIssueLayoutTypes;
  loadError?: boolean;
  onRetry?: () => void;
}

export const IssueLayoutHOC = observer(function IssueLayoutHOC(props: Props) {
  const { layout, loadError = false, onRetry } = props;
  const { t } = useTranslation();

  const storeType = useIssueStoreType();
  const { issues } = useIssues(storeType);

  const issueCount = issues.getGroupIssueCount(undefined, undefined, false);

  if (loadError) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-3 text-center">
        <AlertTriangle className="size-8 text-tertiary" />
        <p className="text-14 text-secondary">{t("something_went_wrong")}</p>
        {onRetry && (
          <Button variant="secondary" size="sm" onClick={() => void onRetry()}>
            {t("common.retry")}
          </Button>
        )}
      </div>
    );
  }

  if (issues?.getIssueLoader() === "init-loader" || issueCount === undefined) {
    return <ActiveLoader layout={layout} />;
  }

  if (issues.getGroupIssueCount(undefined, undefined, false) === 0 && layout !== EIssueLayoutTypes.CALENDAR) {
    return <IssueLayoutEmptyState storeType={storeType} />;
  }

  return <LayoutErrorBoundary key={layout}>{props.children}</LayoutErrorBoundary>;
});
