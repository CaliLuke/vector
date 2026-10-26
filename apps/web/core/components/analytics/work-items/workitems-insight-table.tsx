/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";

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
import { InsightTable } from "../insight-table";
import type { AnalyticsColumnDef, AnalyticsRow } from "../insight-table/table-features";

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
            header: () => <div className="text-left">{columnsLabels["project__name"]}</div>,
            cell: ({ row }) => {
              const project = projectRows?.find((candidate) => candidate.id === row.original.project_id);
              return (
                <div className="flex items-center gap-2">
                  {project?.logo_props ? (
                    <Logo logo={project.logo_props} size={18} />
                  ) : (
                    <ProjectIcon className="h-4 w-4" />
                  )}
                  {project?.name}
                </div>
              );
            },
            meta: {
              export: {
                key: columnsLabels["project__name"],
                value: (row) => row.original.project__name?.toString() ?? "",
              },
            },
          }
        : {
            accessorKey: "display_name",
            header: () => <div className="text-left">{columnsLabels["display_name"]}</div>,
            cell: ({ row }: { row: AnalyticsRow<WorkItemInsightColumns> }) => (
              <div className="text-left">
                <div className="flex items-center gap-2">
                  {row.original.avatar_url && row.original.avatar_url !== "" ? (
                    <Avatar
                      name={row.original.display_name}
                      src={getFileURL(row.original.avatar_url)}
                      size={24}
                      shape="circle"
                    />
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
            ),
            meta: {
              export: {
                key: columnsLabels["display_name"],
                value: (row) => row.original.display_name?.toString() ?? "",
              },
            },
          },
      {
        accessorKey: "backlog_work_items",
        header: () => <div className="text-right">{columnsLabels["backlog_work_items"]}</div>,
        cell: ({ row }) => <div className="text-right">{row.original.backlog_work_items}</div>,
        meta: {
          export: {
            key: columnsLabels["backlog_work_items"],
            value: (row) => row.original.backlog_work_items.toString(),
          },
        },
      },
      {
        accessorKey: "started_work_items",
        header: () => <div className="text-right">{columnsLabels["started_work_items"]}</div>,
        cell: ({ row }) => <div className="text-right">{row.original.started_work_items}</div>,
        meta: {
          export: {
            key: columnsLabels["started_work_items"],
            value: (row) => row.original.started_work_items.toString(),
          },
        },
      },
      {
        accessorKey: "un_started_work_items",
        header: () => <div className="text-right">{columnsLabels["un_started_work_items"]}</div>,
        cell: ({ row }) => <div className="text-right">{row.original.un_started_work_items}</div>,
        meta: {
          export: {
            key: columnsLabels["un_started_work_items"],
            value: (row) => row.original.un_started_work_items.toString(),
          },
        },
      },
      {
        accessorKey: "completed_work_items",
        header: () => <div className="text-right">{columnsLabels["completed_work_items"]}</div>,
        cell: ({ row }) => <div className="text-right">{row.original.completed_work_items}</div>,
        meta: {
          export: {
            key: columnsLabels["completed_work_items"],
            value: (row) => row.original.completed_work_items.toString(),
          },
        },
      },
      {
        accessorKey: "cancelled_work_items",
        header: () => <div className="text-right">{columnsLabels["cancelled_work_items"]}</div>,
        cell: ({ row }) => <div className="text-right">{row.original.cancelled_work_items}</div>,
        meta: {
          export: {
            key: columnsLabels["cancelled_work_items"],
            value: (row) => row.original.cancelled_work_items.toString(),
          },
        },
      },
    ],
    [columnsLabels, isPeekView, projectRows, t]
  );
  return (
    <div className="flex flex-col gap-2">
      {!canRead ? (
        <AnalyticsAccessDenied />
      ) : query.isError ? (
        <AnalyticsQueryError onRetry={() => void query.refetch()} />
      ) : (
        <InsightTable<"work-items">
          analyticsType="work-items"
          data={workItemsData}
          isLoading={isLoading}
          columns={columns}
          columnsLabels={columnsLabels}
          headerText={isPeekView ? t("common.assignee") : t("common.projects")}
          onExport={(rows) => workItemsData && exportCSV(rows, columns, scope.workspaceSlug ?? "")}
        />
      )}
    </div>
  );
};

export default WorkItemsInsightTable;
