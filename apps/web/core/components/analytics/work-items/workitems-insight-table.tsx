/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext, useMemo } from "react";

import { useQuery } from "@tanstack/react-query";
import { UserRound } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import { Logo } from "@plane/propel/emoji-icon-picker";
import { ProjectIcon } from "@plane/propel/icons";
// plane package imports
import type { AnalyticsTableDataMap, WorkItemInsightColumns } from "@plane/types";
// plane web components
import { Avatar } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
// hooks
import { useAnalyticsState } from "@/providers/analytics-state-provider";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { analyticsStatsOptions } from "@/queries/analytics";
// plane web components
import { AnalyticsAccessDenied, AnalyticsQueryError } from "../query-error";
import { exportCSV } from "../export";
import { AnalyticsColumnHeader, AnalyticsValueCell } from "../insight-table/column-renderers";
import { InsightTable } from "../insight-table";
import type { AnalyticsColumnDef, AnalyticsRow } from "../insight-table/table-features";

const COUNT_COLUMNS = [
  "backlog_work_items",
  "started_work_items",
  "un_started_work_items",
  "completed_work_items",
  "cancelled_work_items",
] as const;

const ProjectCatalogueContext = createContext<ReturnType<typeof useWorkspaceProjectDetails>["projects"]>([]);

function ProjectCell({ row }: { row: AnalyticsRow<WorkItemInsightColumns> }) {
  const projectRows = useContext(ProjectCatalogueContext);
  const project = projectRows?.find((candidate) => candidate.id === row.original.project_id);
  return (
    <div className="flex items-center gap-2">
      {project?.logo_props ? <Logo logo={project.logo_props} size={18} /> : <ProjectIcon className="h-4 w-4" />}
      {project?.name}
    </div>
  );
}

function AssigneeCell({ row }: { row: AnalyticsRow<WorkItemInsightColumns> }) {
  const { t } = useTranslation();
  return (
    <div className="text-left">
      <div className="flex items-center gap-2">
        {row.original.avatar_url && row.original.avatar_url !== "" ? (
          <Avatar name={row.original.display_name} src={getFileURL(row.original.avatar_url)} size={24} shape="circle" />
        ) : (
          <div className="flex h-4 w-4 flex-shrink-0 items-center justify-center overflow-hidden rounded-full bg-layer-1 capitalize">
            {row.original.display_name ? (
              row.original.display_name?.[0]
            ) : (
              <UserRound className="text-secondary" size={12} />
            )}
          </div>
        )}
        <span className="break-words text-secondary">{row.original.display_name ?? t(`Unassigned`)}</span>
      </div>
    </div>
  );
}

const WorkItemsInsightTable = function WorkItemsInsightTable() {
  const { t } = useTranslation();
  // store hooks
  const { scope, requestFilters, isPeekView } = useAnalyticsState();
  const projectCatalogue = useWorkspaceProjectDetails(scope.workspaceSlug);
  const projectRows = projectCatalogue.projects;
  const query = useQuery(analyticsStatsOptions(scope, requestFilters, isPeekView));
  const { data: workItemsData, isLoading } = query;
  const canRead = isPeekView ? scope.hasProjectAnalyticsAccess : scope.hasWorkspaceAnalyticsAccess;
  // derived values
  const columnsLabels: Record<keyof Omit<WorkItemInsightColumns, "project_id" | "avatar_url" | "assignee_id">, string> =
    useMemo(
      () => ({
        backlog_work_items: t("workspace_projects.state.backlog"),
        started_work_items: t("workspace_projects.state.started"),
        un_started_work_items: t("workspace_projects.state.unstarted"),
        completed_work_items: t("workspace_projects.state.completed"),
        cancelled_work_items: t("workspace_projects.state.cancelled"),
        project__name: t("common.project"),
        display_name: t("common.assignee"),
      }),
      [t]
    );
  const columns: AnalyticsColumnDef<AnalyticsTableDataMap["work-items"]>[] = useMemo(
    () => [
      !isPeekView
        ? {
            accessorKey: "project__name",
            header: AnalyticsColumnHeader,
            cell: ProjectCell,
            meta: {
              export: {
                key: columnsLabels["project__name"],
                value: (row) => row.original.project__name?.toString() ?? "",
              },
            },
          }
        : {
            accessorKey: "display_name",
            header: AnalyticsColumnHeader,
            cell: AssigneeCell,
            meta: {
              export: {
                key: columnsLabels["display_name"],
                value: (row) => row.original.display_name?.toString() ?? "",
              },
            },
          },
      ...COUNT_COLUMNS.map((key): AnalyticsColumnDef<WorkItemInsightColumns> => ({
        accessorKey: key,
        header: AnalyticsColumnHeader,
        cell: AnalyticsValueCell,
        meta: {
          align: "right",
          export: { key: columnsLabels[key], value: (row) => row.original[key].toString() },
        },
      })),
    ],
    [columnsLabels, isPeekView]
  );
  return (
    <div className="flex flex-col gap-2">
      {!canRead ? (
        <AnalyticsAccessDenied />
      ) : query.isError ? (
        <AnalyticsQueryError onRetry={() => void query.refetch()} />
      ) : (
        <ProjectCatalogueContext.Provider value={projectRows}>
          <InsightTable<"work-items">
            analyticsType="work-items"
            data={workItemsData}
            isLoading={isLoading}
            columns={columns}
            columnsLabels={columnsLabels}
            headerText={isPeekView ? t("common.assignee") : t("common.projects")}
            onExport={(rows) => workItemsData && exportCSV(rows, columns, scope.workspaceSlug ?? "")}
          />
        </ProjectCatalogueContext.Provider>
      )}
    </div>
  );
};

export default WorkItemsInsightTable;
