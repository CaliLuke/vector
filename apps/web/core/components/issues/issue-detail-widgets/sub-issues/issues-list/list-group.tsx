/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

import { CircleDashed } from "lucide-react";
import { ALL_ISSUES } from "@plane/constants";
import { ChevronRightIcon } from "@plane/propel/icons";
import type { IGroupByColumn, TIssue, TIssueServiceType, TSubIssueOperations } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { Collapsible } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { SubIssuesListItem } from "./list-item";

interface TSubIssuesListGroupProps {
  workItems: TIssue[];
  projectId: string;
  workspaceSlug: string;
  group: IGroupByColumn;
  serviceType: TIssueServiceType;
  canEdit: boolean;
  parentIssueId: string;
  rootIssueId: string;
  handleIssueCrudState: (
    key: "create" | "existing" | "update" | "delete",
    issueId: string,
    issue?: TIssue | null
  ) => void;
  subIssueOperations: TSubIssueOperations;
  storeType?: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>;
  spacingLeft?: number;
}

export const SubIssuesListGroup = function SubIssuesListGroup(props: TSubIssuesListGroupProps) {
  const {
    group,
    serviceType,
    canEdit,
    parentIssueId,
    rootIssueId,
    projectId,
    workspaceSlug,
    handleIssueCrudState,
    subIssueOperations,
    workItems,
    storeType = EIssuesStoreType.PROJECT,
    spacingLeft = 0,
  } = props;

  const isAllIssues = group.id === ALL_ISSUES;

  // states
  const [isCollapsibleOpen, setIsCollapsibleOpen] = useState(true);

  if (!workItems.length) return null;

  return (
    <>
      <Collapsible
        isOpen={isCollapsibleOpen}
        onToggle={() => setIsCollapsibleOpen(!isCollapsibleOpen)}
        title={
          !isAllIssues && (
            <div className="flex items-center gap-2 p-3">
              <ChevronRightIcon
                className={cn("size-3.5 text-placeholder transition-all", {
                  "rotate-90": isCollapsibleOpen,
                })}
                strokeWidth={2.5}
              />
              <div className="grid flex-shrink-0 place-items-center overflow-hidden">
                {group.icon ?? <CircleDashed className="size-3.5" strokeWidth={2} />}
              </div>
              <span className="text-13 font-medium text-primary">{group.name}</span>
              <span className="text-13 text-placeholder">{workItems.length}</span>
            </div>
          )
        }
        buttonClassName={cn("hidden", !isAllIssues && "block")}
      >
        {workItems.map((issue) => (
          <SubIssuesListItem
            key={issue.id}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            parentIssueId={parentIssueId}
            rootIssueId={rootIssueId}
            issue={issue}
            canEdit={canEdit}
            handleIssueCrudState={handleIssueCrudState}
            subIssueOperations={subIssueOperations}
            issueServiceType={serviceType}
            spacingLeft={spacingLeft}
            storeType={storeType}
          />
        ))}
      </Collapsible>
    </>
  );
};
