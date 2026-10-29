/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssueLayoutTypes } from "@plane/types";
import { WorkspaceSpreadsheetRoot } from "@/components/issues/issue-layouts/spreadsheet/roots/workspace-root";
import type { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";

export type TWorkspaceLayoutProps = {
  activeLayout: EIssueLayoutTypes | undefined;
  isDefaultView: boolean;
  isLoading?: boolean;
  workspaceSlug: string;
  globalViewId: string;
  routeFilters: {
    [key: string]: string;
  };
  readState: ReturnType<typeof useIssueListPresentation>;
  globalViewsLoading: boolean;
};

export function WorkspaceActiveLayout(props: TWorkspaceLayoutProps) {
  const {
    activeLayout = EIssueLayoutTypes.SPREADSHEET,
    isDefaultView,
    isLoading,
    workspaceSlug,
    globalViewId,
    routeFilters,
    readState,
    globalViewsLoading,
  } = props;
  switch (activeLayout) {
    case EIssueLayoutTypes.SPREADSHEET:
      return (
        <WorkspaceSpreadsheetRoot
          isDefaultView={isDefaultView}
          isLoading={isLoading}
          workspaceSlug={workspaceSlug}
          globalViewId={globalViewId}
          routeFilters={routeFilters}
          readState={readState}
          globalViewsLoading={globalViewsLoading}
        />
      );
    default:
      return <></>;
  }
}
