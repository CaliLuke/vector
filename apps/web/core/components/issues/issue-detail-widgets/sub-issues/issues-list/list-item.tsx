/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

import { Link as Loader } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import { LinkIcon, EditIcon, TrashIcon, CloseIcon, ChevronRightIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssue, TIssueServiceType, TSubIssueOperations } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
import { ControlLink, CustomMenu } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { generateWorkItemLink } from "@plane/utils/work-item";
import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";
import { WithDisplayPropertiesHOC } from "@/components/issues/issue-layouts/properties/with-display-properties-HOC";
import { useSubWorkItemPreferences } from "@/providers/sub-work-item-preferences";
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import useIssuePeekOverviewRedirection from "@/hooks/use-issue-peek-overview-redirection";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
import { SubIssuesListItemProperties } from "./properties";
import { useIssuePropertyChange } from "@/hooks/use-issue-property-change";
import type { SubIssueAction } from "../types";
import { SubIssuesListRoot } from "./root";

type Props = {
  workspaceSlug: string;
  projectId: string;
  parentIssueId: string;
  rootIssueId: string;
  spacingLeft: number;
  canEdit: boolean;
  onAction: (action: SubIssueAction) => void;
  subIssueOperations: TSubIssueOperations;
  issue: TIssue;
  issueServiceType?: TIssueServiceType;
  storeType?: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>;
};

export const SubIssuesListItem = function SubIssuesListItem(props: Props) {
  const {
    workspaceSlug,
    projectId,
    parentIssueId,
    rootIssueId,
    issue,
    spacingLeft = 10,
    canEdit,
    onAction,
    subIssueOperations,
    issueServiceType = EIssueServiceType.ISSUES,
    storeType = EIssuesStoreType.PROJECT,
  } = props;
  const { toggleCreateIssueModal, toggleDeleteIssueModal } = useIssueDetailUiState(issueServiceType);
  const [isExpanded, setExpanded] = useState(false);
  const issueId = issue.id;
  const childQuery = useWorkItemSubItems(
    workspaceSlug,
    issue.project_id ?? undefined,
    issueId,
    issueServiceType,
    isExpanded
  );
  const projectRows = useWorkspaceProjectDetails(workspaceSlug).projects;
  const { handleRedirection } = useIssuePeekOverviewRedirection();
  const { isMobile } = usePlatformOS();
  const projectDetail = projectRows?.find((projectRow) => projectRow.id === issue.project_id);
  const subIssueCount = issue.sub_issues_count ?? 0;
  const subIssuePreferences = useSubWorkItemPreferences({
    workspaceSlug,
    projectId,
    parentId: parentIssueId,
    serviceType: issueServiceType,
  }).preferences;
  const displayProperties = subIssuePreferences?.displayProperties ?? {};
  const isCurrentIssueRoot = issueId === rootIssueId;
  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issue.project_id,
    issueId,
    projectIdentifier: projectDetail?.identifier,
    sequenceId: issue.sequence_id,
  });

  return (
    <div className="relative">
      <div className="group flex min-h-11 w-full items-center gap-2 py-1 pr-2 transition-all hover:bg-surface-2">
        <SubIssueDisclosure
          hasChildren={subIssueCount > 0 && !isCurrentIssueRoot}
          isExpanded={isExpanded}
          isLoading={isExpanded && childQuery.isPending}
          spacingLeft={spacingLeft}
          onToggle={() => setExpanded((current) => !current)}
        />
        <ControlLink
          id={`issue-${issue.id}`}
          href={workItemLink}
          onClick={() => handleRedirection(workspaceSlug, issue, isMobile)}
          className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 truncate"
        >
          <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="key">
            <div className="flex-shrink-0">
              {projectDetail && (
                <IssueIdentifier
                  projectId={projectDetail.id}
                  issueTypeId={issue.type_id}
                  projectIdentifier={projectDetail.identifier}
                  issueSequenceId={issue.sequence_id}
                  size="xs"
                  variant="secondary"
                />
              )}
            </div>
          </WithDisplayPropertiesHOC>
          <Tooltip tooltipContent={issue.name} isMobile={isMobile}>
            <span className="w-0 flex-1 truncate text-13 text-primary">{issue.name}</span>
          </Tooltip>
        </ControlLink>
        <SubIssuesListItemProperties
          workspaceSlug={workspaceSlug}
          parentIssueId={parentIssueId}
          issueId={issueId}
          canEdit={canEdit}
          updateSubIssue={subIssueOperations.updateSubIssue}
          displayProperties={displayProperties}
          issue={issue}
        />
        <SubIssueActions
          workItemLink={workItemLink}
          issue={issue}
          workspaceSlug={workspaceSlug}
          parentIssueId={parentIssueId}
          canEdit={canEdit}
          serviceType={issueServiceType}
          operations={subIssueOperations}
          onEdit={() => {
            onAction({
              kind: "update",
              issue,
              parent: { projectId, issueId: parentIssueId, serviceType: issueServiceType },
            });
            toggleCreateIssueModal(true);
          }}
          onDelete={() => {
            onAction({
              kind: "delete",
              issue,
              parent: { projectId, issueId: parentIssueId, serviceType: issueServiceType },
            });
            toggleDeleteIssueModal(issueId);
          }}
        />
      </div>
      <SubIssueChildren
        isExpanded={isExpanded}
        isLoaded={childQuery.isSuccess}
        issue={issue}
        subIssueCount={subIssueCount}
        isCurrentIssueRoot={isCurrentIssueRoot}
        storeType={storeType}
        workspaceSlug={workspaceSlug}
        rootIssueId={rootIssueId}
        spacingLeft={spacingLeft}
        canEdit={canEdit}
        onAction={onAction}
      />
    </div>
  );
};

function SubIssueDisclosure({
  hasChildren,
  isExpanded,
  isLoading,
  spacingLeft,
  onToggle,
}: {
  hasChildren: boolean;
  isExpanded: boolean;
  isLoading: boolean;
  spacingLeft: number;
  onToggle: () => void;
}) {
  return (
    <div className="flex size-5 flex-shrink-0 items-center justify-center" style={{ marginLeft: spacingLeft - 10 }}>
      {hasChildren &&
        (isLoading ? (
          <div className="flex h-full w-full items-center justify-center rounded-xs bg-layer-1 transition-all">
            <Loader width={14} strokeWidth={2} className="animate-spin" />
          </div>
        ) : (
          <button
            type="button"
            aria-label={isExpanded ? "Collapse sub-items" : "Expand sub-items"}
            className="flex h-full w-full items-center justify-center text-placeholder hover:text-tertiary"
            onClick={onToggle}
          >
            <ChevronRightIcon
              className={cn("size-3.5 transition-all", { "rotate-90": isExpanded })}
              strokeWidth={2.5}
            />
          </button>
        ))}
    </div>
  );
}

function SubIssueChildren({
  isExpanded,
  isLoaded,
  issue,
  subIssueCount,
  isCurrentIssueRoot,
  storeType,
  workspaceSlug,
  rootIssueId,
  spacingLeft,
  canEdit,
  onAction,
}: {
  isExpanded: boolean;
  isLoaded: boolean;
  issue: TIssue;
  subIssueCount: number;
  isCurrentIssueRoot: boolean;
  storeType: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>;
  workspaceSlug: string;
  rootIssueId: string;
  spacingLeft: number;
  canEdit: boolean;
  onAction: Props["onAction"];
}) {
  if (!isExpanded || !isLoaded || !issue.project_id || subIssueCount === 0 || isCurrentIssueRoot) return null;

  return (
    <SubIssuesListRoot
      storeType={storeType}
      workspaceSlug={workspaceSlug}
      projectId={issue.project_id}
      parentIssueId={issue.id}
      rootIssueId={rootIssueId}
      spacingLeft={spacingLeft + 22}
      canEdit={canEdit}
      onAction={onAction}
    />
  );
}

function SubIssueActions({
  issue,
  workItemLink,
  workspaceSlug,
  parentIssueId,
  canEdit,
  serviceType,
  operations,
  onEdit,
  onDelete,
}: {
  issue: TIssue;
  workItemLink: string;
  workspaceSlug: string;
  parentIssueId: string;
  canEdit: boolean;
  serviceType: TIssueServiceType;
  operations: TSubIssueOperations;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const { t } = useTranslation();
  const entity = serviceType === EIssueServiceType.ISSUES ? t("common.sub_work_items") : t("issue.label", { count: 1 });
  const removal = useIssuePropertyChange(
    { workspaceSlug, projectId: issue.project_id, issueId: issue.id, context: `sub-item:${parentIssueId}` },
    canEdit && Boolean(issue.project_id),
    {
      success: t("entity.remove.success", { entity }),
      error: t("entity.remove.failed", { entity }),
    }
  );
  const remove = () => {
    const projectId = issue.project_id;
    if (projectId) removal.run(() => operations.removeSubIssue(workspaceSlug, projectId, parentIssueId, issue.id));
  };

  return (
    <div className="flex-shrink-0 text-13">
      <CustomMenu placement="bottom-end" ellipsis>
        {canEdit && (
          <CustomMenu.MenuItem onClick={onEdit} disabled={removal.pending}>
            <div className="flex items-center gap-2">
              <EditIcon className="h-3.5 w-3.5" strokeWidth={2} />
              <span>{t("issue.edit")}</span>
            </div>
          </CustomMenu.MenuItem>
        )}

        <CustomMenu.MenuItem onClick={() => operations.copyLink(workItemLink)}>
          <div className="flex items-center gap-2">
            <LinkIcon className="h-3.5 w-3.5" strokeWidth={2} />
            <span>{t("issue.copy_link")}</span>
          </div>
        </CustomMenu.MenuItem>

        {canEdit && issue.project_id && (
          <CustomMenu.MenuItem onClick={remove} disabled={removal.disabled}>
            <div className="flex items-center gap-2">
              <CloseIcon className="h-3.5 w-3.5" strokeWidth={2} />
              {serviceType === EIssueServiceType.ISSUES ? t("issue.remove.parent.label") : t("issue.remove.label")}
            </div>
          </CustomMenu.MenuItem>
        )}

        {canEdit && (
          <CustomMenu.MenuItem onClick={onDelete} disabled={removal.pending}>
            <div className="flex items-center gap-2">
              <TrashIcon className="h-3.5 w-3.5" strokeWidth={2} />
              <span>{t("issue.delete.label")}</span>
            </div>
          </CustomMenu.MenuItem>
        )}
      </CustomMenu>
    </div>
  );
}
