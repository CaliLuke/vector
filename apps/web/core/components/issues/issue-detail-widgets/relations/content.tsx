/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo, useState } from "react";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import type { RelationItemAction } from "../../relations/types";

// plane imports
import { useTranslation } from "@plane/i18n";
import type { TIssue, TIssueServiceType, TIssueRelationTypes } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { Collapsible } from "@plane/ui";
// components
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import { useTimeLineRelationOptions } from "@/components/relations";
import { CreateUpdateEpicModal } from "@/components/epic-modal";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useIssueRelations } from "@/hooks/use-issue-resources";
// helper
import { DeleteIssueModal } from "../../delete-issue-modal";
import { RelationIssueList } from "../../relations/issue-list";
import { useRelationOperations } from "./helper";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
  issueServiceType?: TIssueServiceType;
};

type RelationAction = RelationItemAction & {
  source: ReturnType<ReturnType<typeof useIssueRelations>["captureReconciliation"]>;
  scopeKey: string;
};

type TVisibleRelationRow = {
  relationKey: TIssueRelationTypes;
  issues: TIssue[];
  icon: ((size: number) => React.ReactElement) | undefined;
  labelKey: string | undefined;
  className: string | undefined;
};

function getVisibleRelationRows(
  relations: ReturnType<typeof useIssueRelations>["relations"],
  options: ReturnType<typeof useTimeLineRelationOptions>
): TVisibleRelationRow[] {
  if (!relations) return [];
  return (Object.keys(relations) as TIssueRelationTypes[]).flatMap((relationKey) => {
    const option = options[relationKey];
    const issues = relations[relationKey];
    if (!option || issues.length === 0) return [];
    return [{ relationKey, issues, icon: option.icon, labelKey: option.i18n_label, className: option.className }];
  });
}

export type TRelationObject = {
  key: TIssueRelationTypes;
  i18n_label: string;
  className: string;
  icon: (size: number) => React.ReactElement;
  placeholder: string;
};

function RelationRetryButton({
  queryError,
  accessError,
  retry,
  refetch,
}: {
  queryError: boolean;
  accessError: boolean;
  retry: () => Promise<unknown>;
  refetch: () => Promise<unknown>;
}) {
  const { t } = useTranslation();
  if (!queryError && !accessError) return null;
  return (
    <button
      type="button"
      className="text-11 text-danger-primary"
      onClick={() => void (accessError ? retry() : refetch())}
    >
      {t("common.retry")}
    </button>
  );
}

export const RelationsCollapsibleContent = function RelationsCollapsibleContent(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled = false, issueServiceType = EIssueServiceType.ISSUES } = props;
  // plane hooks
  const { t } = useTranslation();
  const [action, setAction] = useState<RelationAction>();
  const { relations, query, isError, retry, captureReconciliation, scope } = useIssueRelations({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: issueServiceType,
  });
  const scopeKey = JSON.stringify(scope);
  const visibleRelations = getVisibleRelationRows(relations, useTimeLineRelationOptions());
  const onAction = (itemAction: RelationItemAction) => {
    if (disabled || !itemAction.issue.project_id) return;
    setAction({ ...itemAction, source: captureReconciliation(), scopeKey });
  };

  return (
    <>
      <RelationRetryButton queryError={query.isError} accessError={isError} retry={retry} refetch={query.refetch} />
      <div className="gap- flex flex-col">
        {visibleRelations.map((relation) => (
          <div key={relation.relationKey}>
            <Collapsible
              buttonClassName="w-full"
              title={
                <div className={`flex h-9 w-full items-center gap-1 px-2.5 py-1 ${relation.className}`}>
                  <span>{relation.icon ? relation.icon(14) : null}</span>
                  <span className="text-13 leading-5 font-medium">{relation.labelKey ? t(relation.labelKey) : ""}</span>
                </div>
              }
              defaultOpen
            >
              <RelationIssueList
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                issueId={issueId}
                relationKey={relation.relationKey}
                issues={relation.issues}
                disabled={disabled}
                onAction={onAction}
                issueServiceType={issueServiceType}
              />
            </Collapsible>
          </div>
        ))}
      </div>

      {action && action.scopeKey === scopeKey && !disabled && (
        <RelationActionModal
          key={JSON.stringify([
            workspaceSlug,
            projectId,
            issueId,
            issueServiceType,
            action.kind,
            action.issue.project_id,
            action.issue.id,
          ])}
          workspaceSlug={workspaceSlug}
          parentServiceType={issueServiceType}
          action={action}
          onClose={() => setAction(undefined)}
        />
      )}
    </>
  );
};

function RelationActionModal({
  workspaceSlug,
  parentServiceType,
  action,
  onClose,
}: {
  workspaceSlug: string;
  parentServiceType: TIssueServiceType;
  action: RelationAction;
  onClose: () => void;
}) {
  const { issue } = action;
  const operations = useRelationOperations(
    workspaceSlug,
    issue.project_id ?? "",
    issue.id,
    issue.is_epic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );
  const acknowledge = useMemo(() => operations.captureAcceptedUpdate(), [operations]);
  const isMounted = useCommittedScope(JSON.stringify([workspaceSlug, issue.project_id, issue.id]));
  const { toggleDeleteIssueModal, toggleCreateIssueModal } = useIssueDetailUiState(parentServiceType);
  if (action.kind === "delete")
    return (
      <DeleteIssueModal
        isOpen
        handleClose={() => {
          onClose();
          toggleDeleteIssueModal(null);
        }}
        data={issue}
        onSubmit={async () => {
          if (!action.source.isCurrent() || !issue.project_id) throw new Error("The work item relation has changed.");
          await operations.remove();
          await action.source.refresh();
        }}
        isEpic={!!issue.is_epic}
      />
    );
  const EditModal = issue.is_epic ? CreateUpdateEpicModal : CreateUpdateIssueModal;
  return (
    <EditModal
      isOpen
      onClose={() => {
        onClose();
        toggleCreateIssueModal(false);
      }}
      data={issue}
      onSubmit={async (record: TIssue) => {
        if (
          !isMounted() ||
          !action.source.isCurrent() ||
          record.id !== issue.id ||
          record.project_id !== issue.project_id
        )
          return;
        await acknowledge(record);
        await action.source.refresh();
      }}
    />
  );
}
