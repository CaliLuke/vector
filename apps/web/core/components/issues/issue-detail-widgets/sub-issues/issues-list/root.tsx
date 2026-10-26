/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useGroupColumns } from "@/components/issues/issue-layouts/use-group-columns";

// plane imports
import { ListFilter } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import type { GroupByColumnTypes, TIssue, TIssueServiceType, TSubIssueOperations } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
// hooks
import { SectionEmptyState } from "@/components/empty-state/section-empty-state-root";
import { isWorkspaceLevel } from "@/components/issues/issue-layouts/utils";
import { useSubWorkItemPreferences } from "@/providers/sub-work-item-preferences";

import { getFilteredWorkItems, getGroupedWorkItemIds } from "@/helpers/issue-list/grouping";
import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";
import { SubIssuesListGroup } from "./list-group";
type Props = {
  workspaceSlug: string;
  projectId: string;
  parentIssueId: string;
  rootIssueId: string;
  spacingLeft: number;
  canEdit: boolean;
  handleIssueCrudState: (
    key: "create" | "existing" | "update" | "delete",
    issueId: string,
    issue?: TIssue | null
  ) => void;
  subIssueOperations: TSubIssueOperations;
  issueServiceType?: TIssueServiceType;
  storeType: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>;
};

export const SubIssuesListRoot = function SubIssuesListRoot(props: Props) {
  const {
    workspaceSlug,
    projectId,
    parentIssueId,
    rootIssueId,
    canEdit,
    handleIssueCrudState,
    subIssueOperations,
    issueServiceType = EIssueServiceType.ISSUES,
    storeType = EIssuesStoreType.PROJECT,
    spacingLeft = 0,
  } = props;
  const { t } = useTranslation();
  // store hooks
  // derived values
  const { preferences: filters, resetPreferences } = useSubWorkItemPreferences({
    workspaceSlug,
    projectId,
    parentId: rootIssueId,
    serviceType: issueServiceType,
  });
  const rootChildren = useWorkItemSubItems(workspaceSlug, projectId, rootIssueId, issueServiceType);
  const nestedChildren = useWorkItemSubItems(
    workspaceSlug,
    projectId,
    parentIssueId,
    issueServiceType,
    parentIssueId !== rootIssueId
  );
  const rootSubIssues = (rootChildren.subItems ?? []).filter((issue) => !issue.archived_at);
  const filteredSubWorkItems = getFilteredWorkItems(rootSubIssues, filters.filters);
  const groupedSubIssues = getGroupedWorkItemIds(
    filteredSubWorkItems,
    filters.displayFilters.group_by,
    filters.displayFilters.order_by
  );
  const isRootLevel = rootIssueId === parentIssueId;
  const group_by = isRootLevel ? (filters?.displayFilters?.group_by ?? null) : null;
  const filteredSubWorkItemsCount = filteredSubWorkItems.length;

  const groups = useGroupColumns({
    workspaceSlug,
    groupBy: group_by as GroupByColumnTypes,
    includeNone: true,
    isWorkspaceLevel: isWorkspaceLevel(storeType),
    isEpic: issueServiceType === EIssueServiceType.EPICS,
    projectId,
  });

  const getWorkItems = (groupId: string) => {
    const source = isRootLevel ? rootSubIssues : (nestedChildren.subItems ?? []);
    const ids = isRootLevel ? (groupedSubIssues[groupId] ?? []) : source.map((item) => item.id);
    const byId = new Map(source.map((item) => [item.id, item]));
    return ids.flatMap((id) => {
      const issue = byId.get(id);
      return issue && !issue.archived_at ? [issue] : [];
    });
  };

  const isSubWorkItems = issueServiceType === EIssueServiceType.ISSUES;

  return (
    <div className="relative">
      {isRootLevel && filteredSubWorkItemsCount === 0 ? (
        <SectionEmptyState
          title={
            !isSubWorkItems
              ? t("sub_work_item.empty_state.list_filters.title")
              : t("sub_work_item.empty_state.sub_list_filters.title")
          }
          description={
            !isSubWorkItems
              ? t("sub_work_item.empty_state.list_filters.description")
              : t("sub_work_item.empty_state.sub_list_filters.description")
          }
          icon={<ListFilter />}
          customClassName={storeType !== EIssuesStoreType.EPIC ? "border-none" : ""}
          actionElement={
            <Button variant="secondary" onClick={resetPreferences}>
              {t("sub_work_item.empty_state.list_filters.action")}
            </Button>
          }
        />
      ) : (
        groups?.map((group) => (
          <SubIssuesListGroup
            key={group.id}
            workItems={getWorkItems(group.id)}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
            group={group}
            serviceType={issueServiceType}
            canEdit={canEdit}
            parentIssueId={parentIssueId}
            rootIssueId={rootIssueId}
            handleIssueCrudState={handleIssueCrudState}
            subIssueOperations={subIssueOperations}
            storeType={storeType}
            spacingLeft={spacingLeft}
          />
        ))
      )}
    </div>
  );
};
